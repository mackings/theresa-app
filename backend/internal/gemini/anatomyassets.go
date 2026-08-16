package gemini

import "sort"

// AnatomyAssets is the closed set of real, properly-licensed 3D anatomy
// models available - individual body-part meshes from BodyParts3D (DBCLS,
// Japan), licensed CC BY-SA 2.1 Japan (see frontend/public/anatomy/ATTRIBUTION.md
// for the full citation/attribution this license requires). Gemini can only
// pick a key from this exact list - enforced via show_3d_model's schema
// enum (see tools.go) - never an arbitrary asset reference. The actual STL
// files are static frontend assets; this map exists purely to validate/
// enumerate the schema on the backend side, kept in sync by hand with
// frontend/src/lib/board/anatomyAssets.ts's own copy (a small, rarely-changing
// list, same spirit as tools.go's codeLanguageAliases).
//
// Heart uses BodyParts3D's "wall of heart" (FMA7274) - the one directly
// downloadable primitive that represents the heart's overall solid shape;
// its finer internal sub-structures (chambers, valves, vessels) are either
// unavailable as standalone meshes in this dataset or would require
// assembling ~40+ small fragments, which isn't worth the payload/complexity
// for what's still a "simplified illustrative" educational view either way.
//
// Brain IS curated, but not at full fragment-set fidelity: a curated 39-file
// subset (~48MB) - every structure actually visible on the brain's outer
// surface (cortex gyri/lobes, cerebellum, brainstem), dropping only the
// deep-internal structures (ventricles, basal ganglia, thalamus,
// hippocampus, amygdala, etc.) that would be invisible inside a solid
// rotate-view model anyway - not an arbitrary size cut, the full 73-fragment
// set is 69MB and was checked directly.
//
// Foot is curated at full fragment-set fidelity (90 files, ~24MB) - real
// geometry, checked directly and found much lighter than brain's despite a
// similarly large fragment count.
//
// Hand and eye were tried and removed - real geometry existed and resolved
// correctly, but didn't display well in the app (too small/hard to read at
// the sizes the board actually renders at, unlike foot and the other
// curated assets) - a presentation-quality call, not a licensing or data
// problem.
//
// This list still excludes a few checked-and-rejected candidates, each for
// a concrete reason (not an oversight): spinal cord has no real tissue mesh
// anywhere in this dataset under any name - confirmed by cross-referencing
// every composite that references it (FMA7647) across the full dataset, not
// just its own entry - the only thing available is the hollow central canal
// inside it, which wouldn't look like a spinal cord at all; skin is a real
// single mesh but a 56MB full-body surface, far too heavy to host; thyroid
// gland, tongue, individual teeth, larynx, uterus, and ovary simply aren't
// present in this dataset at all (it appears to be modeled from a male
// reference body for the reproductive organs, and several soft-tissue
// structures were never captured as standalone meshes).
var AnatomyAssets = map[string]string{
	"liver":           "Liver",
	"kidneys":         "Kidneys",
	"lungs":           "Lungs",
	"heart":           "Heart",
	"stomach":         "Stomach",
	"pancreas":        "Pancreas",
	"spleen":          "Spleen",
	"femur":           "Femur",
	"bladder":         "Bladder",
	"small_intestine": "Small intestine",
	"large_intestine": "Large intestine",
	"humerus":         "Humerus",
	"trachea":         "Trachea",
	"esophagus":       "Esophagus",
	"gallbladder":     "Gallbladder",
	"thymus":          "Thymus",
	"prostate":        "Prostate",
	"testis":          "Testis",
	"clavicle":        "Clavicle",
	"scapula":         "Scapula",
	"sternum":         "Sternum",
	"mandible":        "Mandible (jawbone)",
	"brain":           "Brain",
	"foot":            "Foot",
}

// AnatomyAssetKeys returns AnatomyAssets' keys in a stable (sorted) order -
// used to build the show_3d_model tool's asset_key enum, so the generated
// schema doesn't vary between process restarts (Go map iteration order is
// randomized).
func AnatomyAssetKeys() []string {
	keys := make([]string, 0, len(AnatomyAssets))
	for k := range AnatomyAssets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
