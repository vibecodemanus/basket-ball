package game

import "math"

type Hoop struct {
	// Rim endpoints
	RimLeftX, RimRightX float32
	RimY                float32
	// Backboard
	BackboardX float32
	BackboardTopY, BackboardBottomY float32
	// Scoring zone (below rim)
	NetTopY, NetBottomY float32
}

var (
	LeftHoop = Hoop{
		RimLeftX:  HoopLeftX - RimWidth/2,
		RimRightX: HoopLeftX + RimWidth/2,
		RimY:      HoopY,
		BackboardX:      HoopLeftX - RimWidth/2 - 4,
		BackboardTopY:   HoopY - BackboardHeight/2,
		BackboardBottomY: HoopY + BackboardHeight/2,
		NetTopY:    HoopY,
		NetBottomY: HoopY + 30,
	}
	RightHoop = Hoop{
		RimLeftX:  HoopRightX - RimWidth/2,
		RimRightX: HoopRightX + RimWidth/2,
		RimY:      HoopY,
		BackboardX:      HoopRightX + RimWidth/2 + 4,
		BackboardTopY:   HoopY - BackboardHeight/2,
		BackboardBottomY: HoopY + BackboardHeight/2,
		NetTopY:    HoopY,
		NetBottomY: HoopY + 30,
	}
)

const rimRadius = float32(5)

func CheckBallHoop(b *BallState, h *Hoop, prevY float32) bool {
	// Check rim collision (circle vs circle at rim endpoints)
	checkRimPoint(b, h.RimLeftX, h.RimY)
	checkRimPoint(b, h.RimRightX, h.RimY)

	// Check backboard collision
	checkBackboard(b, h)

	// Check scoring: ball passes downward through the net zone
	if b.InFlight && prevY < h.RimY && b.Y >= h.RimY {
		if b.X > h.RimLeftX+rimRadius && b.X < h.RimRightX-rimRadius {
			return true // SCORE!
		}
	}

	return false
}

func checkRimPoint(b *BallState, rimX, rimY float32) {
	dx := b.X - rimX
	dy := b.Y - rimY
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	minDist := BallRadius + rimRadius

	if dist < minDist && dist > 0 {
		// Normalize
		nx := dx / dist
		ny := dy / dist

		// Separate
		overlap := minDist - dist
		b.X += nx * overlap
		b.Y += ny * overlap

		// Reflect velocity
		dot := b.VX*nx + b.VY*ny
		b.VX -= 2 * dot * nx
		b.VY -= 2 * dot * ny

		// Apply restitution
		b.VX *= RestitutionRim
		b.VY *= RestitutionRim
	}
}

func checkBackboard(b *BallState, h *Hoop) {
	// Determine which side the backboard is on
	isLeft := h.BackboardX < h.RimLeftX

	if b.Y+BallRadius > h.BackboardTopY && b.Y-BallRadius < h.BackboardBottomY {
		if isLeft {
			// Left backboard: ball hits from the right
			if b.X-BallRadius < h.BackboardX && b.X > h.BackboardX-BallRadius*2 {
				b.X = h.BackboardX + BallRadius
				b.VX = -b.VX * RestitutionBackboard
			}
		} else {
			// Right backboard: ball hits from the left
			if b.X+BallRadius > h.BackboardX && b.X < h.BackboardX+BallRadius*2 {
				b.X = h.BackboardX - BallRadius
				b.VX = -b.VX * RestitutionBackboard
			}
		}
	}
}
