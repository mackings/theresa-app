"use client";

import { useEffect, useRef, useState } from "react";
import { Canvas, useFrame } from "@react-three/fiber";
import { Html, Line, OrbitControls } from "@react-three/drei";
import * as THREE from "three";
import { Info } from "lucide-react";
import { STLLoader } from "three/examples/jsm/loaders/STLLoader.js";
import { SCENE3D_DWELL_MS } from "@/lib/board/timing";
import { Scene3D, Scene3DPart } from "@/types/board";
import { ANATOMY_ASSETS, ANATOMY_ATTRIBUTION, ANATOMY_SOURCE_URL } from "@/lib/board/anatomyAssets";

// Converts a hotspot's raw mm-space position through the same
// center/scale transform applied to the asset's mesh group (see
// AnatomyAssetScene) - world position = scale * (raw - center). Hotspots
// render as siblings of the scaled mesh group rather than nested children
// of it, so a marker's own on-screen size stays consistent across assets
// regardless of how small or large that asset's own scale factor is.
function applyAssetTransform(
  position: [number, number, number],
  center: [number, number, number],
  scale: number
): [number, number, number] {
  return [
    (position[0] - center[0]) * scale,
    (position[1] - center[1]) * scale,
    (position[2] - center[2]) * scale,
  ];
}

// A small hoverable labeled landmark point. Depth-tested like any real mesh,
// so a marker placed on the far side of a solid organ is naturally occluded
// by the organ's own geometry - no manual facing/angle math needed. Hover
// reveals the label via Html (plain DOM, CSP-safe - see PartMesh's own note
// on why not drei's Text).
function HotspotMarker({ position, color, label }: { position: [number, number, number]; color: string; label: string }) {
  const [hovered, setHovered] = useState(false);
  const dotRef = useRef<THREE.Mesh>(null);

  useFrame(({ clock }) => {
    if (!dotRef.current) return;
    dotRef.current.scale.setScalar(hovered ? 1.5 : 1 + Math.sin(clock.elapsedTime * 2.2) * 0.12);
  });

  return (
    <group position={position}>
      <mesh
        ref={dotRef}
        onPointerOver={(e) => {
          e.stopPropagation();
          setHovered(true);
        }}
        onPointerOut={(e) => {
          e.stopPropagation();
          setHovered(false);
        }}
      >
        <sphereGeometry args={[0.05, 16, 16]} />
        <meshBasicMaterial color="#ffffff" toneMapped={false} />
      </mesh>
      <mesh scale={0.55}>
        <sphereGeometry args={[0.05, 16, 16]} />
        <meshBasicMaterial color={color} toneMapped={false} depthTest={false} />
      </mesh>
      {hovered && (
        <Html center distanceFactor={8} style={{ pointerEvents: "none" }}>
          <span
            className="whitespace-nowrap rounded-md border px-2 py-1 text-xs font-medium shadow-[var(--shadow-sm)]"
            style={{ borderColor: color, background: "var(--color-surface-raised)", color: "var(--color-text-primary)" }}
          >
            {label}
          </span>
        </Html>
      )}
    </group>
  );
}

// One real anatomy part - loads a real STL file and renders it as-is. All
// files for one asset_key share BodyParts3D's own consistent body
// coordinate system, so siblings just compose correctly with zero manual
// positioning - see anatomyAssets.ts's own comment.
function AnatomyPartMesh({ url, color }: { url: string; color: string }) {
  const [geometry, setGeometry] = useState<THREE.BufferGeometry | null>(null);

  useEffect(() => {
    let cancelled = false;
    new STLLoader().load(url, (geo) => {
      if (cancelled) return;
      geo.computeVertexNormals();
      setGeometry(geo);
    });
    return () => {
      cancelled = true;
    };
  }, [url]);

  if (!geometry) return null;
  return (
    <mesh geometry={geometry}>
      <meshStandardMaterial color={color} roughness={0.6} metalness={0.05} />
    </mesh>
  );
}

// A real, curated anatomy model - one or more real STL files rendered
// together, plus the license-required attribution line. The raw STL
// coordinates are in millimeters within BodyParts3D's own full-body
// coordinate system (far from the origin, far too large for a normal
// Three.js scene) - center/scale (precomputed per asset, see
// anatomyAssets.ts) re-centers and resizes the whole group at once, so
// multi-file assets like lungs/kidneys keep their correct relative
// positions to each other.
function AnatomyAssetScene({ assetKey }: { assetKey: string }) {
  const asset = ANATOMY_ASSETS[assetKey];
  if (!asset) return null;

  const [cx, cy, cz] = asset.center;
  const s = asset.scale;

  return (
    <group>
      <group position={[-cx * s, -cy * s, -cz * s]} scale={s}>
        {asset.files.map((url) => (
          <AnatomyPartMesh key={url} url={url} color="#b5524a" />
        ))}
      </group>
      {(asset.hotspots ?? []).map((h) => (
        <HotspotMarker
          key={h.id}
          position={applyAssetTransform(h.position, asset.center, s)}
          color={h.color}
          label={h.label}
        />
      ))}
    </group>
  );
}

const PART_COLOR_FALLBACK = "#5fa694";

function PartMesh({ part }: { part: Scene3DPart }) {
  const size = part.size && part.size > 0 ? part.size : 1;
  const color = part.color || PART_COLOR_FALLBACK;

  const geometryArgs: [number, number, number] = [size, size, size];

  return (
    <group position={[part.x, part.y, part.z]}>
      <mesh>
        {part.shape === "box" && <boxGeometry args={geometryArgs} />}
        {part.shape === "cylinder" && <cylinderGeometry args={[size / 2, size / 2, size, 24]} />}
        {part.shape === "cone" && <coneGeometry args={[size / 2, size, 24]} />}
        {part.shape === "torus" && <torusGeometry args={[size / 2, size / 4, 16, 32]} />}
        {part.shape === "capsule" && <capsuleGeometry args={[size / 3, size / 2, 8, 16]} />}
        {(part.shape === "sphere" || !part.shape) && <sphereGeometry args={[size / 2, 24, 24]} />}
        <meshStandardMaterial color={color} roughness={0.4} metalness={0.1} />
      </mesh>
      {/* Html (plain DOM, CSS-positioned) instead of drei's Text - Text
          (troika-three-text) generates SDF glyphs via a Web Worker, which
          this app's CSP blocks (no worker-src directive), silently breaking
          the whole 3D render tree. Html needs no worker at all. */}
      <Html position={[0, size / 2 + 0.3, 0]} center distanceFactor={8}>
        <span className="whitespace-nowrap rounded bg-white/80 px-1 text-[10px] font-medium text-[#333333]">
          {part.label}
        </span>
      </Html>
    </group>
  );
}

function LinkLine({ from, to }: { from: Scene3DPart; to: Scene3DPart }) {
  const points: [number, number, number][] = [
    [from.x, from.y, from.z],
    [to.x, to.y, to.z],
  ];
  return <Line points={points} color="#999999" lineWidth={1} />;
}

function ProceduralScene({ scene3d }: { scene3d: Scene3D }) {
  const parts = scene3d.parts ?? [];
  const byLabel = new Map(parts.map((p) => [p.label, p]));

  return (
    <group>
      {parts.map((part, i) => (
        <PartMesh key={i} part={part} />
      ))}
      {(scene3d.links ?? []).map((link, i) => {
        const from = byLabel.get(link.from);
        const to = byLabel.get(link.to);
        if (!from || !to) return null;
        return <LinkLine key={i} from={from} to={to} />;
      })}
    </group>
  );
}

// ThreeDBoard renders a "3d" board unit - either a real curated anatomy
// asset (scene3d.asset_key) or a small procedural scene (scene3d.parts/links).
// Same onComplete contract as DiagramBoard/CodeBoard: a ref-stabilized
// onCompleteRef, a fixed dwell before calling onComplete - which does NOT
// unmount the scene, since Board.tsx accumulates completed units, so
// OrbitControls stays interactive for as long as the board stays on screen.
export function ThreeDBoard({
  title,
  scene3d,
  onComplete,
}: {
  title?: string;
  scene3d: Scene3D;
  onComplete: () => void;
}) {
  const onCompleteRef = useRef(onComplete);
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  useEffect(() => {
    const timeoutId = setTimeout(() => onCompleteRef.current(), SCENE3D_DWELL_MS);
    return () => clearTimeout(timeoutId);
  }, [scene3d]);

  const isRealAsset = !!scene3d.asset_key && !!ANATOMY_ASSETS[scene3d.asset_key];

  return (
    <div className="px-6 py-6">
      {title && (
        <p className="mb-3 font-mono text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
          {title}
        </p>
      )}
      {scene3d.caption && (
        <p className="mb-2 text-sm text-[var(--color-text-secondary)]">{scene3d.caption}</p>
      )}
      <div className="relative h-[32rem] overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)]">
        <Canvas camera={{ position: [0, 0, 6], fov: 45 }}>
          <ambientLight intensity={0.7} />
          <directionalLight position={[5, 8, 5]} intensity={1} />
          <directionalLight position={[-5, -3, -5]} intensity={0.3} />
          {isRealAsset ? (
            <AnatomyAssetScene assetKey={scene3d.asset_key!} />
          ) : (
            <ProceduralScene scene3d={scene3d} />
          )}
          <OrbitControls enablePan={false} />
        </Canvas>
        {/* Small, unobtrusive credit icon rather than a full visible text
            line - BodyParts3D's CC BY-SA 2.1 Japan license requires credit
            wherever the model is shown, but doesn't require it to be
            prominent; a hover/click-accessible icon satisfies that without
            being visual clutter in the teaching UI. */}
        {isRealAsset && (
          <a
            href={ANATOMY_SOURCE_URL}
            target="_blank"
            rel="noopener noreferrer"
            title={ANATOMY_ATTRIBUTION}
            aria-label={ANATOMY_ATTRIBUTION}
            className="absolute bottom-1.5 right-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-[var(--color-surface-raised)]/80 text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
          >
            <Info className="h-3 w-3" />
          </a>
        )}
      </div>
    </div>
  );
}
