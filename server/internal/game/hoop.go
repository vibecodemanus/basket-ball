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
	// ── 1. Check scoring FIRST — before collisions modify position.
	// A clean shot through the hoop must score before rim physics
	// can accidentally push the ball out of the scoring zone.
	if b.InFlight && prevY < h.RimY && b.Y >= h.RimY {
		if b.X > h.RimLeftX+rimRadius && b.X < h.RimRightX-rimRadius {
			return true // SCORE!
		}
	}

	// ── 2. Rim collision (circle vs circle at each rim endpoint)
	checkRimPoint(b, h.RimLeftX, h.RimY)
	checkRimPoint(b, h.RimRightX, h.RimY)

	// ── 3. Backboard collision
	checkBackboard(b, h)

	return false
}

func checkRimPoint(b *BallState, rimX, rimY float32) {
	dx := b.X - rimX
	dy := b.Y - rimY
	distSq := dx*dx + dy*dy
	minDist := BallRadius + rimRadius

	if distSq < minDist*minDist && distSq > 0.001 {
		dist := float32(math.Sqrt(float64(distSq)))

		// Normal from rim to ball
		nx := dx / dist
		ny := dy / dist

		// Only reflect if ball is moving toward the rim point.
		// This prevents double-bounce jitter when the ball is
		// separating after a previous collision.
		dot := b.VX*nx + b.VY*ny
		if dot >= 0 {
			// Ball is already moving away — just separate, don't reflect.
			overlap := minDist - dist
			if overlap > 0 {
				b.X += nx * overlap
				b.Y += ny * overlap
			}
			return
		}

		// Separate
		overlap := minDist - dist
		b.X += nx * overlap
		b.Y += ny * overlap

		// Reflect velocity along collision normal
		b.VX -= 2 * dot * nx
		b.VY -= 2 * dot * ny

		// Apply restitution
		b.VX *= RestitutionRim
		b.VY *= RestitutionRim
	}
}

func checkBackboard(b *BallState, h *Hoop) {
	// Skip if ball is not in the Y range of the backboard
	if b.Y+BallRadius <= h.BackboardTopY || b.Y-BallRadius >= h.BackboardBottomY {
		return
	}

	isLeft := h.BackboardX < h.RimLeftX
	// Backboard visual is 4px wide; use 2px half-width for collision
	const bbHalf = float32(2)

	if isLeft {
		bbRight := h.BackboardX + bbHalf
		// Ball must be moving left (toward backboard) and overlapping the right face.
		// Without the velocity check, balls behind the backboard teleport through.
		if b.VX < 0 && b.X-BallRadius < bbRight && b.X > h.BackboardX {
			b.X = bbRight + BallRadius
			b.VX = float32(math.Abs(float64(b.VX))) * RestitutionBackboard
		}
	} else {
		bbLeft := h.BackboardX - bbHalf
		// Ball must be moving right (toward backboard) and overlapping the left face.
		if b.VX > 0 && b.X+BallRadius > bbLeft && b.X < h.BackboardX {
			b.X = bbLeft - BallRadius
			b.VX = -float32(math.Abs(float64(b.VX))) * RestitutionBackboard
		}
	}
}
