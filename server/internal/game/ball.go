package game

import (
	"log"
	"math"
	"math/rand"
)

func NewBall() BallState {
	return BallState{
		X:          CourtWidth / 2,
		Y:          FloorY - BallRadius,
		Owner:      -1,
		ShooterIdx: -1,
	}
}

func clampF(val, min, max float32) float32 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func StepBall(b *BallState, players *[2]PlayerState) {
	if b.Owner >= 0 {
		// Ball follows the holder
		p := &players[b.Owner]
		offsetX := float32(p.Facing) * 14
		b.X = p.X + offsetX
		b.Y = p.Y + 8
		b.VX = 0
		b.VY = 0
		return
	}

	// Decrement pickup cooldown
	if b.PickupCooldown > 0 {
		b.PickupCooldown--
	}

	// Increment shot age
	if b.InFlight && b.ShotAgeTicks < 255 {
		b.ShotAgeTicks++
	}

	if b.InFlight || b.Owner == -1 {
		// Gravity
		b.VY += Gravity * DT

		// Integrate
		b.X += b.VX * DT
		b.Y += b.VY * DT

		// Floor bounce
		if b.Y+BallRadius >= FloorY {
			b.Y = FloorY - BallRadius
			b.VY = -b.VY * RestitutionFloor
			b.VX *= 0.95 // friction

			if float32(math.Abs(float64(b.VY))) < 20 {
				b.VY = 0
				b.InFlight = false
			}
		}

		// Side walls
		if b.X-BallRadius < 0 {
			b.X = BallRadius
			b.VX = -b.VX * 0.8
		}
		if b.X+BallRadius > CourtWidth {
			b.X = CourtWidth - BallRadius
			b.VX = -b.VX * 0.8
		}

		// Ceiling
		if b.Y-BallRadius < 0 {
			b.Y = BallRadius
			b.VY = -b.VY * 0.5
		}

		// Ball-player interception (AABB vs circle)
		if b.InFlight {
			CheckBallPlayerCollision(b, players)
		}
	}

	// Pickup check (only when ball is free, cooldown expired, and not flying fast)
	if b.Owner == -1 && b.PickupCooldown == 0 && (!b.InFlight || (math.Abs(float64(b.VX)) < 100 && math.Abs(float64(b.VY)) < 100)) {
		for i := range players {
			p := &players[i]
			dx := p.X - b.X
			dy := p.Y - b.Y
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if dist < PlayerWidth/2+BallRadius+4 {
				b.Owner = int8(i)
				b.InFlight = false
				b.ShooterIdx = -1
				p.HasBall = true
				return
			}
		}
	}
}

// shotAccuracy returns the probability (0.15..0.6) that a shot hits the hoop,
// based on the shooter's distance from the opponent's hoop.
// Under the hoop (dist ~0): 0.6
// At the 3-point line (dist = ThreePointRadius): 0.25
// At center court (max range): 0.15
// Linear interpolation between zones.
func shotAccuracy(playerX float32, playerIdx int8) float64 {
	var hoopX float32
	if playerIdx == 0 {
		hoopX = HoopRightX
	} else {
		hoopX = HoopLeftX
	}

	dist := math.Abs(float64(playerX - hoopX))
	threeP := float64(ThreePointRadius)

	if dist <= threeP {
		// Inside 3-point line: lerp 0.6 (under hoop) → 0.25 (at 3pt line)
		t := dist / threeP // 0 at hoop, 1 at 3pt line
		return 0.6 - t*0.35
	}

	// Beyond 3-point line: lerp 0.25 → 0.15 over remaining court distance
	maxDist := float64(CourtWidth) - float64(hoopX)
	if playerIdx == 1 {
		maxDist = float64(hoopX)
	}
	remaining := maxDist - threeP
	if remaining < 1 {
		return 0.15
	}
	t := (dist - threeP) / remaining // 0 at 3pt line, 1 at far wall
	t = math.Min(1, math.Max(0, t))
	return 0.25 - t*0.1
}

// ShootBall — server auto-calculates angle/force to hit opponent's hoop.
// playerIdx: 0 shoots at right hoop, 1 shoots at left hoop.
// Shot accuracy depends on distance: guaranteed on opponent's half, probabilistic on own half.
func ShootBall(b *BallState, p *PlayerState, playerIdx int8) {
	// Determine target hoop
	var hoopX float32
	if playerIdx == 0 {
		hoopX = HoopRightX
	} else {
		hoopX = HoopLeftX
	}
	hoopY := HoopY

	// Accuracy check — miss means offset target
	accuracy := shotAccuracy(p.X, playerIdx)
	hit := rand.Float64() < accuracy

	targetX := hoopX
	targetY := hoopY
	if !hit {
		// Offset target so ball misses (±35..65px horizontal, ±10..25px vertical)
		offsetX := float32(35 + rand.Float64()*30)
		if rand.Intn(2) == 0 {
			offsetX = -offsetX
		}
		offsetY := float32(-25 + rand.Float64()*35) // -25 to +10
		targetX += offsetX
		targetY += offsetY
		log.Printf("MISS: accuracy=%.2f, offsetX=%.1f offsetY=%.1f", accuracy, offsetX, offsetY)
	}

	// Ball starting position (same as StepBall follow logic)
	startX := p.X + float32(p.Facing)*14
	startY := p.Y + 8

	dx := float64(targetX - startX)
	dy := float64(startY - targetY) // positive = target above
	D := math.Abs(dx)
	H := dy

	var angle, force float64

	if D > 1 {
		// Minimum force: v_min² = g * (H + sqrt(H² + D²))
		rangeHyp := math.Sqrt(H*H + D*D)
		vMinSq := float64(Gravity) * (H + rangeHyp)
		vMin := MinShootForce
		if vMinSq > 0 {
			vMin = float32(math.Sqrt(vMinSq))
		}

		// Comfortable arc: v_min * 1.15
		force = float64(clampF(vMin*1.15, MinShootForce, MaxShootForce))

		// Solve for launch angle: c*u² - D*u + (c + H) = 0 where c = g*D²/(2*v²), u = tan(θ)
		v := force
		c := float64(Gravity) * D * D / (2 * v * v)
		discriminant := D*D - 4*c*(c+H)

		if discriminant >= 0 {
			// Higher arc solution for natural basketball trajectory
			u := (D + math.Sqrt(discriminant)) / (2 * c)
			launchAngle := math.Atan(u) // 0..π/2
			if dx >= 0 {
				angle = launchAngle
			} else {
				angle = math.Pi - launchAngle
			}
		} else {
			// Target out of range — max-range angle
			lineOfSight := math.Atan2(H, D)
			bestAngle := math.Max(math.Pi/4, (math.Pi/4+lineOfSight)/2+0.2)
			if dx >= 0 {
				angle = bestAngle
			} else {
				angle = math.Pi - bestAngle
			}
		}
	} else {
		// Directly above — shoot straight up
		angle = math.Pi / 2
		force = float64(clampF(float32(math.Abs(dy)*2), MinShootForce, MaxShootForce))
	}

	// Clamp angle to upward arc only
	angle = math.Max(0.1, math.Min(math.Pi-0.1, angle))

	b.VX = float32(force * math.Cos(angle))
	b.VY = -float32(force * math.Sin(angle))
	log.Printf("SHOOT: playerIdx=%d hit=%v accuracy=%.2f angle=%.3f force=%.1f → VX=%.1f VY=%.1f (player %.1f → hoop %.1f)",
		playerIdx, hit, accuracy, angle, force, b.VX, b.VY, p.X, hoopX)
	b.X = startX
	b.Y = startY
	b.Owner = -1
	b.InFlight = true
	b.PickupCooldown = 30 // ~0.5 seconds before ball can be picked up
	b.ShooterIdx = playerIdx
	b.ShotAgeTicks = 0
	b.ShotOriginX = p.X // record for 3-point detection
	p.HasBall = false
	p.Anim = AnimShoot
}

// CheckBallPlayerCollision — AABB (player body) vs Circle (ball) collision.
// Deflects ball off defender's body during flight.
// Shooter can't collide with own shot for first 30 ticks.
func CheckBallPlayerCollision(b *BallState, players *[2]PlayerState) {
	for i := range players {
		// Skip shooter for first 30 ticks
		if b.ShooterIdx == int8(i) && b.ShotAgeTicks < 30 {
			continue
		}

		p := &players[i]
		// Player AABB: centered at (p.X, p.Y), size PlayerWidth x PlayerHeight
		halfW := PlayerWidth / 2
		halfH := PlayerHeight / 2
		pLeft := p.X - halfW
		pRight := p.X + halfW
		pTop := p.Y - halfH
		pBottom := p.Y + halfH

		// Find closest point on AABB to ball center
		closestX := clampF(b.X, pLeft, pRight)
		closestY := clampF(b.Y, pTop, pBottom)

		dx := b.X - closestX
		dy := b.Y - closestY
		distSq := dx*dx + dy*dy

		if distSq < BallRadius*BallRadius {
			// Collision! Deflect ball
			dist := float32(math.Sqrt(float64(distSq)))
			if dist < 0.001 {
				dist = 0.001
			}

			// Normal from player to ball
			nx := dx / dist
			ny := dy / dist

			// Push ball out of player
			overlap := BallRadius - dist
			b.X += nx * overlap
			b.Y += ny * overlap

			// Reflect velocity along normal with deflection multiplier
			dot := b.VX*nx + b.VY*ny
			b.VX = (b.VX - 2*dot*nx) * DeflectSpeedMult
			b.VY = (b.VY - 2*dot*ny) * DeflectSpeedMult

			// Reset shooter (ball is now deflected, anyone can pick it up)
			b.ShooterIdx = -1
			b.PickupCooldown = 15 // short cooldown after deflection

			log.Printf("INTERCEPT: player %d deflected ball at (%.1f,%.1f) → VX=%.1f VY=%.1f", i, b.X, b.Y, b.VX, b.VY)
			return
		}
	}
}

// TryBlockShot checks if a blocker can block a shooter's attempt.
// Requirements: blocker in jump (not grounded), within BlockRange, blocker.Y <= shooter.Y + 10
func TryBlockShot(b *BallState, shooter *PlayerState, shooterIdx int8, blocker *PlayerState) bool {
	if blocker.Grounded {
		return false
	}

	// Distance check
	dx := shooter.X - blocker.X
	dy := shooter.Y - blocker.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist > BlockRange {
		return false
	}

	// Blocker must be at or above shooter's height (with tolerance)
	if blocker.Y > shooter.Y+10 {
		return false
	}

	// Block! Ball flies down and to the side (away from hoop)
	var deflectVX float32
	if shooterIdx == 0 {
		// Shooter aimed right → deflect left
		deflectVX = -200
	} else {
		// Shooter aimed left → deflect right
		deflectVX = 200
	}

	b.VX = deflectVX
	b.VY = 300 // downward
	b.X = shooter.X
	b.Y = shooter.Y
	b.Owner = -1
	b.InFlight = true
	b.PickupCooldown = 15
	b.ShooterIdx = -1
	b.ShotAgeTicks = 0
	shooter.HasBall = false
	blocker.Anim = AnimBlock

	log.Printf("BLOCK: shooter %d blocked by defender at (%.1f,%.1f)", shooterIdx, blocker.X, blocker.Y)
	return true
}
