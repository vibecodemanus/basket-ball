package game

func NewPlayer(x, y float32, facing int8) PlayerState {
	return PlayerState{
		X:        x,
		Y:        y,
		Facing:   facing,
		Grounded: true,
		Anim:     AnimIdle,
	}
}

func ApplyInput(p *PlayerState, input PlayerInput) {
	// Defender (no ball) moves faster; air control is reduced
	speed := PlayerSpeedWithBall
	if !p.HasBall {
		speed = DefenderSpeedBoost
	}
	if !p.Grounded {
		speed *= AirControlMult
	}
	p.VX = float32(input.MoveX) * speed

	if input.MoveX != 0 {
		p.Facing = input.MoveX
	}

	if input.Jump && p.Grounded {
		if p.HasBall {
			p.VY = JumpVelocity
		} else {
			p.VY = DefenderJumpVelocity
		}
		p.Grounded = false
	}
}

func StepPlayer(p *PlayerState) {
	if !p.Grounded {
		p.VY += Gravity * DT
	}

	p.X += p.VX * DT
	p.Y += p.VY * DT

	// Floor collision
	feetY := p.Y + PlayerHeight/2
	if feetY >= FloorY {
		p.Y = FloorY - PlayerHeight/2
		p.VY = 0
		p.Grounded = true
	}

	// Court bounds
	halfW := PlayerWidth / 2
	if p.X-halfW < 0 {
		p.X = halfW
	}
	if p.X+halfW > CourtWidth {
		p.X = CourtWidth - halfW
	}

	// Animation
	if !p.Grounded {
		p.Anim = AnimJump
	} else if p.HasBall && p.VX != 0 {
		p.Anim = AnimDribble
	} else if p.VX != 0 {
		p.Anim = AnimRun
	} else {
		p.Anim = AnimIdle
	}
}
