package prompts

const prototypeDirectionSheetRules = `- Treat the selected perspective as authoritative. Use its backend-derived direction count and layout exactly as defined below.
- Side-on perspective: render exactly 2 direction views in a 1 row x 2 column sheet, with one view in each equal-sized cell.
- SIDE-ON SCALE LOCK — critical: treat the left and right cells as one shared coordinate system at exactly the same camera distance and zoom. Before adding details, reserve an identical invisible subject-safe box in both cells.
- In a side-on sheet, the subject's core body must occupy the same pixel height and equivalent pixel width in both cells. Put the topmost core-body point on the same row, the bottommost contact point on the same row, and the subject centre on the same cell coordinates.
- Keep identical transparent or matte padding above, below, left, and right of the core subject in both side-on cells. Do not enlarge one view to fill apparent empty space, and do not shrink one view because its silhouette, seam pattern, pose, or internal details differ.
- For side-on left and right views, change the facing direction only; never treat the second cell as an independently framed composition. Preserve one shared scale, baseline, centre, pose, and camera. Asymmetric accessories may differ locally, but they must not change the scale of the core body.
- Before finalizing a side-on sheet, compare the two occupied subject bounds: the core-body height, centre, baseline, and surrounding padding must match. If they do not match, redraw the mismatched view rather than compensating with a different zoom.
- Top-down perspective: render exactly 4 direction views in a 2 row x 2 column sheet, with one view in each equal-sized cell.
- Isometric perspective: render exactly 8 direction views in a 2 row x 4 column sheet, with one view in each equal-sized cell.
- Fill cells in normal reading order: left to right across the first row, then left to right across each following row. Complete the first row before starting the second row.
- The zero-based array index is the direction identity used later when an animation selects its prototype reference image. The cell order is therefore mandatory, not illustrative: never reorder, mirror, omit, or duplicate views.
- For 2 directions, use this exact array order: index 0 = left, index 1 = right.
- For 4 directions, use this exact array order: index 0 = front, index 1 = right, index 2 = back, index 3 = left.
- For 8 directions, use this exact array order: index 0 = front, index 1 = front-right, index 2 = right, index 3 = back-right, index 4 = back, index 5 = back-left, index 6 = left, index 7 = front-left.
- Keep the direction sequence internally consistent, but do not add direction names or labels inside the image. The pipeline identifies directions only by their reading-order indexes, which are these zero-based array indexes.
- Use one shared camera, zoom factor, subject scale, and cell-local coordinate system for the entire sheet. Establish one common subject-safe box and baseline before drawing any direction; every direction must remain registered to that guide.
- Each perspective intentionally produces one regular output sheet containing all required views. This is not a collage: do not add labels, separators, frames, scenery, or unrelated content.`
