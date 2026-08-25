package prompts

import (
	"fmt"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

// prototypeLogicalPixelRules tells the image model about the final sprite grid,
// not just the much larger provider canvas. Post-processing keeps the shared
// animation safety margin, so the visible subject has a smaller real
// pixel budget than the nominal output dimensions.
func prototypeLogicalPixelRules(dimensions assetdomain.Size, subject string) string {
	width, height := int(dimensions.Width), int(dimensions.Height)
	margin := max(1, min(width, height)*3/16)
	innerWidth := max(1, width-2*margin)
	innerHeight := max(1, height-2*margin)

	// Detail tiering must follow the drawable logical grid. A nominal 32x32
	// prototype with the canonical animation margin is actually a 20x20 sprite;
	// treating it as a 32px sprite encourages detail that cannot survive
	// final-size reduction.
	tierRule := logicalPixelTierRule(innerWidth, innerHeight, subject)
	subjectRule := `- Express identifying materials and construction using a few high-contrast color clusters rather than fine texture.
- If the object depends on internal linework such as a structural division, opening, rim, spoke, panel boundary, or seam, keep only the indispensable identity-bearing paths. Draw each retained path as one simplified, continuous, consistently coloured logical-pixel line. Do not render it as broken dots, doubled parallel bands, several antialiased shades, or a thick stripe.
- For an elongated object such as a weapon, tool, pole, staff, spear, rod, banner, or other long prop, do not compress the whole design into a thin centred line or fit it into a square silhouette. Use the available drawable area along its long axis, keep the complete functional end and the grip/shaft as one connected readable silhouette, and simplify ornaments before shrinking the main form. Reserve enough short-axis pixels for the functional end to remain unmistakable.`
	smallCanvasRule := ""
	if subject == "character" {
		subjectRule = "- Make the face symbolic and readable: use a small number of high-contrast pixel clusters for eyes, hairline, skin, or a defining mask; omit eyelashes, nostrils, individual teeth, and other sub-pixel facial rendering unless the final grid can represent them as whole pixels."
	}
	if min(width, height) <= 32 {
		smallCanvasRule = `- The requested final canvas has a short edge of 32 pixels or less. Explicitly lower the design detail level before rendering: use fewer, larger connected shapes and broader color clusters. Omit secondary texture, seams, folds, small accessories, tiny construction parts, logos, and decorative marks that cannot remain readable as whole native-size pixels. Preserve an indispensable identity-bearing internal division only as a simplified continuous one-logical-pixel path with one consistent high-contrast colour. Do not create a detailed high-resolution design and expect downscaling or post-processing to simplify it.`
		if subject == "character" {
			smallCanvasRule += ` Simplify facial anatomy, clothing construction, equipment ornament, and limb separation before rendering; preserve identity through silhouette, proportion, and a few major color regions.`
		}
	}

	return fmt.Sprintf(`Final logical-pixel budget:
- Design each direction view for an actual final canvas of %d x %d pixels, not for the provider's high-resolution render canvas.
- Post-processing crops the visible subject and fits it into at most %d x %d drawable logical pixels while retaining the fixed %d-pixel safety margin used by prototype and animation frames. Do not add extra padding to imitate this margin.
- Choose complexity from the %d x %d drawable region, not from the nominal %d x %d canvas. Details that only look readable on the provider's large render will be destroyed during final-size reduction.
- Treat every final logical pixel as one deliberate square color block. Keep a uniform logical pixel grid across the entire subject.
- Every important feature must survive on that final grid as whole pixels. Do not invent lines, highlights, facial marks, texture, or gaps thinner than one final logical pixel; make defining features at least two logical pixels wide where the budget permits.
- Use flat color clusters with hard boundaries and strong value separation. Do not encode form through smooth shading, internal gradients, transparency fades, or high-resolution micro-detail that will be averaged away.
%s
%s
%s`, width, height, innerWidth, innerHeight, margin, innerWidth, innerHeight, width, height, smallCanvasRule, subjectRule, tierRule)
}

func logicalPixelTierRule(width, height int, subject string) string {
	shortEdge := min(width, height)
	if subject == "character" {
		switch {
		case shortEdge <= 12:
			return fmt.Sprintf(`- This is an emblem-scale character with only %d x %d drawable logical pixels: reduce it to an iconic silhouette with only head, torso, limb direction, and one identifying color accent. Omit facial features and all costume texture unless represented by a single indispensable high-contrast pixel.
- Use broad connected masses rather than anatomically thin parts. Do not create isolated decorative pixels or one-pixel color noise.`, width, height)
		case shortEdge <= 24:
			return fmt.Sprintf(`- This is an ultra-small character with only %d x %d drawable logical pixels. Design it as a compact game sprite, not as a detailed illustration that will later be shrunk.
- Build the entire character from a few large connected regions: silhouette/outline, skin or face, hair or headwear, main upper-body color, main lower-body color, and at most one essential equipment or identity accent.
- Use no more than about 6-8 visually distinct color roles for the whole character at this tier, and reuse the same base and shade colors across body parts instead of creating a new shade for every small region.
- Limit each material to a base cluster plus at most one broad shadow or highlight cluster. Avoid scattered single-pixel highlights, noise, dithering, fabric folds, muscle definition, laces, seams, lettering, logos, jewellery, and tiny costume trim.
- Keep the face to one readable skin cluster plus at most one or two intentional high-contrast marks. Do not attempt a nose, mouth, both detailed eyes, or other miniature anatomy when those marks would compete for the same pixels.
- Do not put internal shading or texture inside any body part narrower than three logical pixels; keep that part as one flat connected cluster.
- Exaggerate head, torso, hands, feet, and limb thickness enough to remain readable. Avoid one-pixel-wide torsos or limbs where a two-pixel-wide cluster can preserve the silhouette, especially in side views.
- At this tier, silhouette readability overrides anatomical separation: hands may merge into arm clusters and shoes may merge into leg clusters when separate pixels would create noisy fragments.
- Front, back, and side directions must be distinguished primarily by silhouette and large color placement, never by tiny interior marks.`, width, height)
		case shortEdge <= 40:
			return fmt.Sprintf(`- This is a small character with %d x %d drawable logical pixels: prioritize silhouette, head/body separation, and a few broad identity accents; simplify anatomy and surface detail aggressively.
- Keep thin limbs and equipment at least two logical pixels thick where possible, and prefer connected shade clusters over scattered highlights.`, width, height)
		case shortEdge <= 80:
			return fmt.Sprintf("- This is a medium-size character with %d x %d drawable logical pixels: allow several controlled highlight and shadow clusters, while keeping all contours stepped and every feature readable without zooming.", width, height)
		case shortEdge <= 160:
			return fmt.Sprintf("- This character has %d x %d drawable logical pixels: permit more material and costume separation, but retain coarse clustered shading and avoid illustration-scale micro-detail.", width, height)
		default:
			return "- Even at this larger drawable target, construct the character from deliberate coarse pixel clusters and preserve native-size readability rather than high-definition illustration detail."
		}
	}

	switch {
	case shortEdge <= 12:
		return fmt.Sprintf("- This is an emblem-scale object with only %d x %d drawable logical pixels: reduce it to an iconic silhouette, one dominant base region, and only one indispensable identifying accent.", width, height)
	case shortEdge <= 24:
		return fmt.Sprintf("- This is an ultra-small object with only %d x %d drawable logical pixels: use an iconic silhouette, one dominant light region, one dominant shadow region, and only the most essential identifying accents.", width, height)
	case shortEdge <= 40:
		return fmt.Sprintf("- This is a small object with %d x %d drawable logical pixels: prioritize silhouette, major construction, and a few 1-2 pixel accents; simplify surface detail aggressively.", width, height)
	case shortEdge <= 80:
		return fmt.Sprintf("- This is a medium-size object with %d x %d drawable logical pixels: allow several controlled highlight and shadow clusters, while keeping all contours stepped and every feature readable without zooming.", width, height)
	case shortEdge <= 160:
		return fmt.Sprintf("- This object has %d x %d drawable logical pixels: permit more material and construction separation, but retain coarse clustered shading and avoid illustration-scale micro-detail.", width, height)
	default:
		return "- Even at this larger drawable target, construct the object from deliberate coarse pixel clusters and preserve native-size readability rather than high-definition illustration detail."
	}
}
