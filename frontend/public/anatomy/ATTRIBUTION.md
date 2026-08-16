# Anatomy model attribution

The 3D anatomy models in this directory are individual body-part meshes from
**BodyParts3D** (version 3.0 / 20110915), published by the Database Center
for Life Science (DBCLS), Japan.

License: **Creative Commons Attribution-ShareAlike 2.1 Japan**
https://creativecommons.org/licenses/by-sa/2.1/jp/deed.en

Required credit (displayed in-app wherever these models render, per license
terms - see `ThreeDBoard.tsx`):

> BodyParts3D, © The Database Center for Life Science licensed under
> CC Attribution-ShareAlike 2.1 Japan

Source archive: http://dbarchive.biosciencedbc.jp/en/bodyparts3d/download.html
Citation: Mitsuhashi N, Fujieda K, Tamura T, Kawamoto S, Takagi T, Okubo K.
"BodyParts3D: 3D structure database for anatomical concepts."
Nucleic Acids Res. 2009 Jan;37(Database issue):D782-5.
https://doi.org/10.1093/nar/gkn613

## Files in this directory

| File | Anatomical part | Source FMA ID |
|---|---|---|
| `liver.stl` | Liver | FMA7197 |
| `kidney-left.stl` | Left kidney | FMA7205 |
| `kidney-right.stl` | Right kidney | FMA7204 |
| `lung-right-upper.stl` | Upper lobe of right lung | FMA7333 |
| `lung-right-middle.stl` | Middle lobe of right lung | FMA7383 |
| `lung-right-lower.stl` | Lower lobe of right lung | FMA7337 |
| `lung-left-upper.stl` | Upper lobe of left lung | FMA7370 |
| `lung-left-lower.stl` | Lower lobe of left lung | FMA7371 |
| `heart.stl` | Wall of heart (overall heart shape) | FMA7274 |
| `stomach.stl` | Stomach | FMA7148 |
| `pancreas.stl` | Pancreas | FMA7198nsn |
| `spleen.stl` | Spleen | FMA7196 |
| `femur-left.stl` | Left femur | FMA24475 |
| `femur-right.stl` | Right femur | FMA24474 |
| `bladder.stl` | Urinary bladder | FMA15900 |
| `small-intestine-duodenum.stl` | Duodenum | FMA7206 |
| `small-intestine-jejunum.stl` | Jejunum | FMA7207 |
| `small-intestine-ileum.stl` | Ileum | FMA7208 |
| `large-intestine-colon.stl` | Colon | FMA14543nsn |
| `large-intestine-taenia-mesocolic.stl` | Mesocolic taenia (colon muscle band) | FMA76891 |
| `large-intestine-taenia-free.stl` | Free taenia (colon muscle band) | FMA76893 |
| `large-intestine-taenia-omental.stl` | Omental taenia (colon muscle band) | FMA76892 |
| `humerus-left.stl` | Left humerus | FMA23131 |
| `humerus-right.stl` | Right humerus | FMA23130 |
| `trachea.stl` | Trachea | FMA7394 |
| `esophagus.stl` | Esophagus | FMA7131 |
| `gallbladder.stl` | Gallbladder | FMA7202 |
| `thymus-left.stl` | Left lobe of thymus | FMA71195 |
| `thymus-right.stl` | Right lobe of thymus | FMA71194 |
| `prostate.stl` | Prostate | FMA9600 |
| `testis-left.stl` | Left testis | FMA7212 |
| `testis-right.stl` | Right testis | FMA7211 |
| `clavicle-left.stl` | Left clavicle | FMA13323 |
| `clavicle-right.stl` | Right clavicle | FMA13322 |
| `scapula-left.stl` | Left scapula | FMA13396 |
| `scapula-right.stl` | Right scapula | FMA13395 |
| `sternum-manubrium.stl` | Manubrium of sternum | FMA7486 |
| `sternum-body.stl` | Body of sternum | FMA7487 |
| `sternum-xiphoid.stl` | Xiphoid process of sternum | FMA7488 |
| `mandible.stl` | Mandible (jawbone) | FMA52748 |

One more asset is made of many small individual fragment files - too many
to usefully list one row per file here, so it's documented as a group
instead:

- **`brain-*.stl`** (39 files) - a curated subset of BodyParts3D's real
  brain fragments: every structure actually visible on the brain's outer
  surface (both cerebral hemispheres' cortex - all lobes' gyri, both
  insulae - plus the cerebellum, pons, midbrain, and medulla oblongata).
  Deliberately excludes the ~34 deep-internal fragments (ventricles, basal
  ganglia, thalamus, hippocampus, amygdala, and similar) that would be
  invisible inside a solid rotate-view model anyway - not a fidelity
  compromise, just excluding what a viewer could never see from outside.
  Each filename is `brain-<FMA-or-BP-id>.stl`; the full id-to-name mapping
  lives in BodyParts3D's own `parts_list_e.txt`.
- **`foot-*.stl`** (90 files) - every real fragment for both feet together
  (individual tarsal bones, intrinsic muscles, tendons), at full fidelity -
  checked directly and found to total only ~24MB despite the large fragment
  count, so no curation was needed.

The "lungs", "kidneys", "femur", "small intestine", "large intestine",
"humerus", "thymus", "testis", "clavicle", "scapula", "sternum", "brain",
and "foot" concepts are each rendered as a group of these individual part
files together, sharing BodyParts3D's own consistent body coordinate
system - no manual repositioning needed, since all files come from the
same source model.

A few other candidates were tried and removed, or never added: hand and eye
both had real, correctly-licensed geometry (eye in particular - "eyeball",
FMA12513, a genuinely separate real primitive from the "eye" *composite*,
which only decomposes to eyelid-muscle fragments) but didn't display well
in the app at the sizes the board actually renders at, so both were pulled
back out - a presentation-quality call, not a licensing or data problem.
Spinal cord has no real tissue mesh anywhere in this dataset under any
name - confirmed by cross-referencing every composite that references it
(FMA7647) across the full dataset, not just its own entry - only the
hollow central canal inside it exists, which wouldn't look like a spinal
cord at all. Skin is a real single mesh but a 56MB full-body surface, far
too heavy to host. Thyroid gland, tongue, individual teeth, larynx,
uterus, and ovary aren't present in this dataset at all.
