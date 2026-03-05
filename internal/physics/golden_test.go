package physics

import (
	"fmt"
	"testing"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/pkg/constants"
)

type goldenBody struct {
	Name     string
	Position math3d.Vec3
	Velocity math3d.Vec3
}

var golden100 = []goldenBody{
	{Name: "Mercury", Position: math3d.Vec3{X: -3.733485003240434e+12, Y: -1.486702344837168e+12, Z: 2.212333181068592e+11}, Velocity: math3d.Vec3{X: -5.260533845984870e+06, Y: -2.283302455939171e+06, Z: 2.963207680632109e+05}},
	{Name: "Venus", Position: math3d.Vec3{X: -1.059617828606540e+11, Y: -1.894810134660855e+10, Z: 5.857927298583682e+09}, Velocity: math3d.Vec3{X: 5.892491117106738e+03, Y: -3.481517492700944e+04, Z: -8.160152937537715e+02}},
	{Name: "Earth", Position: math3d.Vec3{X: -1.398307921130410e+11, Y: -5.405512214459718e+10, Z: 5.418068737190455e+03}, Velocity: math3d.Vec3{X: 1.025746878009957e+04, Y: -2.790031111102345e+04, Z: 1.550628322379992e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -1.012374913856298e+11, Y: 2.207376163492142e+11, Z: 7.113574189663074e+09}, Velocity: math3d.Vec3{X: -1.963062480788476e+04, Y: -7.702858229803740e+03, Z: 3.211729035417416e+02}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.886604573422185e+11, Y: -2.102854333384294e+11, Z: 1.850888163605164e+10}, Velocity: math3d.Vec3{X: 3.066141660598574e+03, Y: -1.142345944786658e+04, Z: -2.139084290456633e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.105396554482515e+12, Y: -9.854052634147042e+11, Z: -2.677311622744309e+10}, Velocity: math3d.Vec3{X: 5.636999130292052e+03, Y: 6.897301851543280e+03, Z: -3.442151892672511e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 4.431381727355819e+11, Y: 2.830268371248493e+12, Z: 4.774331040633696e+09}, Velocity: math3d.Vec3{X: -6.773197869073106e+03, Y: 7.454511769167975e+02, Z: 9.061926028416924e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.454138684495947e+12, Y: 2.457262250262041e+11, Z: -1.076945101836191e+11}, Velocity: math3d.Vec3{X: -3.501343996575448e+02, Y: 5.507949464187409e+03, Z: -1.053454235813424e+02}},
}

var golden1000 = []goldenBody{
	{Name: "Mercury", Position: math3d.Vec3{X: -3.777554569637604e+13, Y: -1.639617032255346e+13, Z: 2.127865856146297e+12}, Velocity: math3d.Vec3{X: -5.251897315105844e+06, Y: -2.304438677587306e+06, Z: 2.938014174736872e+05}},
	{Name: "Venus", Position: math3d.Vec3{X: 6.882571958622504e+10, Y: -8.651815295566597e+10, Z: -5.156017502439809e+09}, Velocity: math3d.Vec3{X: 2.702915849687493e+04, Y: 2.124340124228639e+04, Z: -1.269983033824565e+03}},
	{Name: "Earth", Position: math3d.Vec3{X: 9.242583153286852e+09, Y: -1.517819832726656e+11, Z: 3.787010219487504e+05}, Velocity: math3d.Vec3{X: 2.925220990403072e+04, Y: 1.700024591318397e+03, Z: 4.900013434607366e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -2.016734034354550e+11, Y: 1.329473629540580e+11, Z: 7.743143704599060e+09}, Velocity: math3d.Vec3{X: -1.033628661954419e+04, Y: -1.854087208699865e+04, Z: -1.343731866347247e+02}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.647876605656676e+11, Y: -2.831064252748589e+11, Z: 1.827573564990028e+10}, Velocity: math3d.Vec3{X: 4.296245602260459e+03, Y: -1.103277445555740e+04, Z: -5.051903326492205e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.140963506320149e+12, Y: -9.398766688020944e+11, Z: -2.897997384992683e+10}, Velocity: math3d.Vec3{X: 5.338619894480343e+03, Y: 7.152940623399410e+03, Z: -3.368095464679706e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 3.991969417452578e+11, Y: 2.834762662711685e+12, Z: 5.360954639855659e+09}, Velocity: math3d.Vec3{X: -6.788647117468896e+03, Y: 6.416120533478812e+02, Z: 9.043375172026551e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.451729944264044e+12, Y: 2.814096275642669e+11, Z: -1.083737589726755e+11}, Velocity: math3d.Vec3{X: -3.933014942836099e+02, Y: 5.505388574897752e+03, Z: -1.042981328875607e+02}},
}

func testGoldenBaseline(t *testing.T, steps int, expected []goldenBody) {
	t.Helper()
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = true
	sim.ShowTrails = false

	for i := 0; i < steps; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	if len(sim.Planets) != len(expected) {
		t.Fatalf("expected %d planets, got %d", len(expected), len(sim.Planets))
	}

	for i, exp := range expected {
		got := sim.Planets[i]
		if got.Name != exp.Name {
			t.Errorf("planet %d: expected name %q, got %q", i, exp.Name, got.Name)
			continue
		}

		assertVec3Near(t, got.Position, exp.Position, 1e-1)
		assertVec3Near(t, got.Velocity, exp.Velocity, 1e-6)
	}
}

func TestGoldenBaseline100(t *testing.T) {
	testGoldenBaseline(t, 100, golden100)
}

func TestGoldenBaseline1000(t *testing.T) {
	testGoldenBaseline(t, 1000, golden1000)
}

func TestGenerateGoldenBaseline(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = true
	sim.ShowTrails = false

	for i := 0; i < 100; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	fmt.Println("// Golden values after 100 steps:")
	for _, p := range sim.Planets {
		fmt.Printf("{Name: %q, Position: math3d.Vec3{X: %.15e, Y: %.15e, Z: %.15e}, Velocity: math3d.Vec3{X: %.15e, Y: %.15e, Z: %.15e}},\n",
			p.Name, p.Position.X, p.Position.Y, p.Position.Z, p.Velocity.X, p.Velocity.Y, p.Velocity.Z)
	}

	for i := 0; i < 900; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	fmt.Println("\n// Golden values after 1000 steps:")
	for _, p := range sim.Planets {
		fmt.Printf("{Name: %q, Position: math3d.Vec3{X: %.15e, Y: %.15e, Z: %.15e}, Velocity: math3d.Vec3{X: %.15e, Y: %.15e, Z: %.15e}},\n",
			p.Name, p.Position.X, p.Position.Y, p.Position.Z, p.Velocity.X, p.Velocity.Y, p.Velocity.Z)
	}
}
