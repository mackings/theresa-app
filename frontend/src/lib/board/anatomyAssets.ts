// The frontend half of the curated real-anatomy registry - keyed
// identically to the backend's gemini.AnatomyAssets (kept in sync by hand,
// a small rarely-changing list). Each key maps to a group of real STL files
// (see public/anatomy/ATTRIBUTION.md for provenance/license) - "lungs" and
// "kidneys" are each several individual part files sharing BodyParts3D's
// own consistent body coordinate system, rendered together as siblings with
// no manual repositioning needed.
export interface AnatomyHotspot {
  id: string;
  label: string;
  // Same raw millimeter, full-body coordinate system as the STL geometry
  // itself (NOT pre-scaled/centered) - hand-placed by rendering the model
  // and visually checking where each point actually lands, then converted
  // through the same center/scale transform as the mesh at render time (see
  // ThreeDBoard.tsx's applyAssetTransform) so it never drifts relative to
  // the surface it's meant to label.
  position: [number, number, number];
  color: string;
}

export interface AnatomyAsset {
  label: string;
  files: string[];
  // BodyParts3D's raw coordinates are in millimeters, positioned within a
  // full-body coordinate system (e.g. z~1000-1400 for upper-torso organs) -
  // nowhere near a usable Three.js scene scale/origin. center/scale are
  // precomputed (see the combined bounding box of ALL of this asset's files
  // together, not per-file, so multi-file assets like lungs/kidneys keep
  // their correct relative positions to each other) so the group renders
  // centered at the origin and sized to fit the camera. Applied as:
  // position = [-center[i]*scale, ...], scale = scale (uniform).
  center: [number, number, number];
  scale: number;
  // Optional labeled landmark points, hover-to-reveal (see HotspotMarker in
  // ThreeDBoard.tsx) - only authored for a handful of the most-asked-about
  // organs so far, not every curated asset.
  hotspots?: AnatomyHotspot[];
}

export const ANATOMY_ASSETS: Record<string, AnatomyAsset> = {
  liver: {
    label: "Liver",
    files: ["/anatomy/liver.stl"],
    center: [-18.47, -117.42, 1117.34],
    scale: 0.0201,
    // Hotspot positions below are real points on the mesh surface itself
    // (found by raycasting the actual triangle geometry from the default
    // camera, not eyeballed) - a first guess based on the bounding box
    // alone landed inside the solid and was invisible (depth-occluded by
    // the organ's own front surface), so every hotspot in this file was
    // re-derived this way and re-verified with a real screenshot.
    hotspots: [
      { id: "right-lobe", label: "Right lobe (larger)", position: [-62.3, -87.3, 1200.07], color: "#ee7c6a" },
      { id: "left-lobe", label: "Left lobe (smaller)", position: [38.14, -134.0, 1183.84], color: "#f2a33b" },
    ],
  },
  kidneys: {
    label: "Kidneys",
    // kidney-left.stl/kidney-right.stl were mislabeled earlier this session
    // (contents swapped relative to their real FMA source) - fixed by
    // renaming the files to match their actual FMA7204/FMA7205 identity
    // (see ATTRIBUTION.md), confirmed here by real per-file centroid X:
    // BodyParts3D's convention is positive X = anatomical left, negative
    // X = anatomical right (cross-checked against the far-left-sided spleen
    // and stomach, both strongly positive X).
    files: ["/anatomy/kidney-left.stl", "/anatomy/kidney-right.stl"],
    center: [-5.18, -71.34, 1058.59],
    scale: 0.02214,
    hotspots: [
      { id: "left-kidney", label: "Left kidney", position: [45.4, -72.74, 1114.84], color: "#f2a33b" },
      { id: "right-kidney", label: "Right kidney", position: [-57.19, -69.08, 1097.58], color: "#ee7c6a" },
    ],
  },
  lungs: {
    label: "Lungs",
    files: [
      "/anatomy/lung-right-upper.stl",
      "/anatomy/lung-right-middle.stl",
      "/anatomy/lung-right-lower.stl",
      "/anatomy/lung-left-upper.stl",
      "/anatomy/lung-left-lower.stl",
    ],
    center: [-1.35, -106.96, 1244.06],
    scale: 0.01545,
    hotspots: [
      { id: "right-lung", label: "Right lung", position: [-44.3, -78.61, 1367.95], color: "#ee7c6a" },
      { id: "left-lung", label: "Left lung", position: [46.05, -78.52, 1367.14], color: "#f2a33b" },
    ],
  },
  heart: {
    label: "Heart",
    files: ["/anatomy/heart.stl"],
    center: [21.42, -121.85, 1237.01],
    scale: 0.03463,
    hotspots: [
      { id: "base", label: "Base of the heart", position: [12.37, -79.74, 1270.88], color: "#f2a33b" },
      { id: "great-vessels", label: "Great vessels", position: [-34.08, -110.47, 1239.44], color: "#6393d8" },
      { id: "apex", label: "Apex of the heart", position: [21.42, -163.6, 1203.02], color: "#ee7c6a" },
    ],
  },
  stomach: {
    label: "Stomach",
    files: ["/anatomy/stomach.stl"],
    center: [43.33, -137.93, 1121.77],
    scale: 0.02744,
    hotspots: [
      { id: "fundus", label: "Fundus", position: [64.18, -97.26, 1179.93], color: "#ee7c6a" },
      { id: "body", label: "Body of stomach", position: [43.33, -136.27, 1169.44], color: "#f2a33b" },
      { id: "pylorus", label: "Pylorus", position: [-8.35, -155.42, 1095.66], color: "#6393d8" },
    ],
  },
  pancreas: {
    label: "Pancreas",
    files: ["/anatomy/pancreas.stl"],
    center: [4.06, -129.5, 1057.91],
    scale: 0.04292,
  },
  spleen: {
    label: "Spleen",
    files: ["/anatomy/spleen.stl"],
    center: [79.46, -97.84, 1112.98],
    scale: 0.04016,
  },
  femur: {
    label: "Femur",
    files: ["/anatomy/femur-left.stl", "/anatomy/femur-right.stl"],
    center: [-0.68, -78.22, 622.99],
    scale: 0.01129,
  },
  bladder: {
    label: "Bladder",
    files: ["/anatomy/bladder.stl"],
    center: [-0.66, -97.3, 815.3],
    scale: 0.04934,
  },
  small_intestine: {
    label: "Small intestine",
    files: [
      "/anatomy/small-intestine-duodenum.stl",
      "/anatomy/small-intestine-jejunum.stl",
      "/anatomy/small-intestine-ileum.stl",
    ],
    center: [3.08, -142.73, 963.28],
    scale: 0.01709,
  },
  large_intestine: {
    label: "Large intestine",
    files: [
      "/anatomy/large-intestine-colon.stl",
      "/anatomy/large-intestine-taenia-mesocolic.stl",
      "/anatomy/large-intestine-taenia-free.stl",
      "/anatomy/large-intestine-taenia-omental.stl",
    ],
    center: [1.9, -125.17, 966.7],
    scale: 0.01719,
  },
  humerus: {
    label: "Humerus",
    files: ["/anatomy/humerus-left.stl", "/anatomy/humerus-right.stl"],
    center: [-0.66, -71.95, 1193.19],
    scale: 0.01046,
  },
  trachea: {
    label: "Trachea",
    files: ["/anatomy/trachea.stl"],
    center: [-0.56, -98.3, 1358.03],
    scale: 0.04698,
  },
  esophagus: {
    label: "Esophagus",
    files: ["/anatomy/esophagus.stl"],
    center: [2.85, -85.9, 1290.92],
    scale: 0.02356,
  },
  gallbladder: {
    label: "Gallbladder",
    files: ["/anatomy/gallbladder.stl"],
    center: [-54.89, -151.18, 1081.76],
    scale: 0.08239,
  },
  thymus: {
    label: "Thymus",
    files: ["/anatomy/thymus-left.stl", "/anatomy/thymus-right.stl"],
    center: [21.36, -143.92, 1289.36],
    scale: 0.03668,
  },
  prostate: {
    label: "Prostate",
    files: ["/anatomy/prostate.stl"],
    center: [-0.66, -83.04, 781.55],
    scale: 0.11501,
  },
  testis: {
    label: "Testis",
    files: ["/anatomy/testis-left.stl", "/anatomy/testis-right.stl"],
    center: [-1.54, -152.32, 704.36],
    scale: 0.08617,
  },
  clavicle: {
    label: "Clavicle",
    files: ["/anatomy/clavicle-left.stl", "/anatomy/clavicle-right.stl"],
    center: [-0.67, -96.99, 1349.31],
    scale: 0.01894,
  },
  scapula: {
    label: "Scapula",
    files: ["/anatomy/scapula-left.stl", "/anatomy/scapula-right.stl"],
    center: [-0.66, -43.86, 1280.67],
    scale: 0.01548,
  },
  sternum: {
    label: "Sternum",
    files: [
      "/anatomy/sternum-manubrium.stl",
      "/anatomy/sternum-body.stl",
      "/anatomy/sternum-xiphoid.stl",
    ],
    center: [-0.66, -167.52, 1257.64],
    scale: 0.02978,
  },
  mandible: {
    label: "Mandible (jawbone)",
    files: ["/anatomy/mandible.stl"],
    center: [-0.67, -139.24, 1475.58],
    scale: 0.03977,
  },
  brain: {
    label: "Brain",
    // A curated subset, not the full 73-fragment dataset (69MB) - keeps
    // every structure actually visible on the brain's outer surface (all
    // cortex gyri/lobes, cerebellum, brainstem: 39 files, ~48MB) and drops
    // only the deep-internal structures (ventricles, basal ganglia,
    // thalamus, hippocampus, amygdala, etc.) that would be invisible
    // inside a solid rotate-view model anyway, not a size-only cut.
    files: [
      "/anatomy/brain-BP49.stl",
      "/anatomy/brain-BP50.stl",
      "/anatomy/brain-FMA61993nsn.stl",
      "/anatomy/brain-FMA62004.stl",
      "/anatomy/brain-FMA62394.stl",
      "/anatomy/brain-FMA67943.stl",
      "/anatomy/brain-FMA67944.stl",
      "/anatomy/brain-FMA72653.stl",
      "/anatomy/brain-FMA72654.stl",
      "/anatomy/brain-FMA72655.stl",
      "/anatomy/brain-FMA72656.stl",
      "/anatomy/brain-FMA72661.stl",
      "/anatomy/brain-FMA72662.stl",
      "/anatomy/brain-FMA72665.stl",
      "/anatomy/brain-FMA72666.stl",
      "/anatomy/brain-FMA72667.stl",
      "/anatomy/brain-FMA72668.stl",
      "/anatomy/brain-FMA72669.stl",
      "/anatomy/brain-FMA72670.stl",
      "/anatomy/brain-FMA72685.stl",
      "/anatomy/brain-FMA72686.stl",
      "/anatomy/brain-FMA72687.stl",
      "/anatomy/brain-FMA72688.stl",
      "/anatomy/brain-FMA72689.stl",
      "/anatomy/brain-FMA72690.stl",
      "/anatomy/brain-FMA72701.stl",
      "/anatomy/brain-FMA72702.stl",
      "/anatomy/brain-FMA72705.stl",
      "/anatomy/brain-FMA72706.stl",
      "/anatomy/brain-FMA72717.stl",
      "/anatomy/brain-FMA72718.stl",
      "/anatomy/brain-FMA72800.stl",
      "/anatomy/brain-FMA72801.stl",
      "/anatomy/brain-FMA72804.stl",
      "/anatomy/brain-FMA72805.stl",
      "/anatomy/brain-FMA72975.stl",
      "/anatomy/brain-FMA72976.stl",
      "/anatomy/brain-FMA72977.stl",
      "/anatomy/brain-FMA72978.stl",
    ],
    center: [-0.66, -89.57, 1554.71],
    scale: 0.02246,
  },
  foot: {
    label: "Foot",
    // Both feet together (90 real bone/muscle/tendon fragments, ~24MB).
    files: [
      "/anatomy/foot-FMA230986.stl",
      "/anatomy/foot-FMA230988.stl",
      "/anatomy/foot-FMA24482.stl",
      "/anatomy/foot-FMA24483.stl",
      "/anatomy/foot-FMA24497.stl",
      "/anatomy/foot-FMA24498.stl",
      "/anatomy/foot-FMA24500.stl",
      "/anatomy/foot-FMA24501.stl",
      "/anatomy/foot-FMA24507.stl",
      "/anatomy/foot-FMA24508.stl",
      "/anatomy/foot-FMA24509.stl",
      "/anatomy/foot-FMA24510.stl",
      "/anatomy/foot-FMA24511.stl",
      "/anatomy/foot-FMA24512.stl",
      "/anatomy/foot-FMA24513.stl",
      "/anatomy/foot-FMA24514.stl",
      "/anatomy/foot-FMA24515.stl",
      "/anatomy/foot-FMA24516.stl",
      "/anatomy/foot-FMA24521.stl",
      "/anatomy/foot-FMA24522.stl",
      "/anatomy/foot-FMA24523.stl",
      "/anatomy/foot-FMA24524.stl",
      "/anatomy/foot-FMA24525.stl",
      "/anatomy/foot-FMA24526.stl",
      "/anatomy/foot-FMA24528.stl",
      "/anatomy/foot-FMA24529.stl",
      "/anatomy/foot-FMA32634.stl",
      "/anatomy/foot-FMA32635.stl",
      "/anatomy/foot-FMA32636.stl",
      "/anatomy/foot-FMA32637.stl",
      "/anatomy/foot-FMA32638.stl",
      "/anatomy/foot-FMA32639.stl",
      "/anatomy/foot-FMA32640.stl",
      "/anatomy/foot-FMA32641.stl",
      "/anatomy/foot-FMA32642.stl",
      "/anatomy/foot-FMA32643.stl",
      "/anatomy/foot-FMA32644.stl",
      "/anatomy/foot-FMA32645.stl",
      "/anatomy/foot-FMA32646.stl",
      "/anatomy/foot-FMA32647.stl",
      "/anatomy/foot-FMA32650.stl",
      "/anatomy/foot-FMA32651.stl",
      "/anatomy/foot-FMA32652.stl",
      "/anatomy/foot-FMA32653.stl",
      "/anatomy/foot-FMA32654.stl",
      "/anatomy/foot-FMA32655.stl",
      "/anatomy/foot-FMA32656.stl",
      "/anatomy/foot-FMA32657.stl",
      "/anatomy/foot-FMA32658.stl",
      "/anatomy/foot-FMA32659.stl",
      "/anatomy/foot-FMA37459.stl",
      "/anatomy/foot-FMA37460.stl",
      "/anatomy/foot-FMA37461.stl",
      "/anatomy/foot-FMA37462.stl",
      "/anatomy/foot-FMA37463.stl",
      "/anatomy/foot-FMA37464.stl",
      "/anatomy/foot-FMA37465.stl",
      "/anatomy/foot-FMA37466.stl",
      "/anatomy/foot-FMA37471.stl",
      "/anatomy/foot-FMA37472.stl",
      "/anatomy/foot-FMA37483.stl",
      "/anatomy/foot-FMA37484.stl",
      "/anatomy/foot-FMA37485.stl",
      "/anatomy/foot-FMA37486.stl",
      "/anatomy/foot-FMA37717.stl",
      "/anatomy/foot-FMA37718.stl",
      "/anatomy/foot-FMA37719.stl",
      "/anatomy/foot-FMA37720.stl",
      "/anatomy/foot-FMA37741.stl",
      "/anatomy/foot-FMA37742.stl",
      "/anatomy/foot-FMA37743.stl",
      "/anatomy/foot-FMA37744.stl",
      "/anatomy/foot-FMA37745.stl",
      "/anatomy/foot-FMA37746.stl",
      "/anatomy/foot-FMA43253.stl",
      "/anatomy/foot-FMA43254.stl",
      "/anatomy/foot-FMA45971.stl",
      "/anatomy/foot-FMA45972.stl",
      "/anatomy/foot-FMA45973.stl",
      "/anatomy/foot-FMA45974.stl",
      "/anatomy/foot-FMA46018.stl",
      "/anatomy/foot-FMA46019.stl",
      "/anatomy/foot-FMA46020.stl",
      "/anatomy/foot-FMA46021.stl",
      "/anatomy/foot-FMA51142.stl",
      "/anatomy/foot-FMA51143.stl",
      "/anatomy/foot-FMA51144.stl",
      "/anatomy/foot-FMA51145.stl",
      "/anatomy/foot-FMA86034.stl",
      "/anatomy/foot-FMA86035.stl",
    ],
    center: [-0.67, -120.01, 31.58],
    scale: 0.01604,
  },
};

// The exact credit text BodyParts3D's CC BY-SA 2.1 Japan license requires
// whenever these models are shown - see public/anatomy/ATTRIBUTION.md.
export const ANATOMY_ATTRIBUTION =
  "Anatomy model: BodyParts3D, © DBCLS, CC BY-SA 2.1 Japan";
export const ANATOMY_SOURCE_URL =
  "http://dbarchive.biosciencedbc.jp/en/bodyparts3d/download.html";
