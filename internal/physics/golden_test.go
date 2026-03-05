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
	{Name: "Mercury", Position: math3d.Vec3{X: -2.998050661944295e+10, Y: 3.837189492087230e+10, Z: 5.886361551271410e+09}, Velocity: math3d.Vec3{X: -4.823982841293429e+04, Y: -2.807767571049469e+04, Z: 2.134089252808137e+03}},
	{Name: "Venus", Position: math3d.Vec3{X: -1.059535260586938e+11, Y: -1.882757140275921e+10, Z: 5.859097978034842e+09}, Velocity: math3d.Vec3{X: 5.905421788031348e+03, Y: -3.465063653003762e+04, Z: -8.145128923200105e+02}},
	{Name: "Earth", Position: math3d.Vec3{X: -1.398307919395429e+11, Y: -5.405512214573059e+10, Z: 5.391504882168951e+03}, Velocity: math3d.Vec3{X: 1.025746920074424e+04, Y: -2.790031100819793e+04, Z: 1.545934384382751e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -1.022683467428483e+11, Y: 2.204221664061728e+11, Z: 7.132305515363196e+09}, Velocity: math3d.Vec3{X: -2.106041993144292e+04, Y: -8.139496669647214e+03, Z: 3.471718871151554e+02}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.885520918318459e+11, Y: -2.107096405709939e+11, Z: 1.850821009814218e+10}, Velocity: math3d.Vec3{X: 3.216642382150143e+03, Y: -1.201261145643650e+04, Z: -2.232351899768032e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.105566423231038e+12, Y: -9.851994769592957e+11, Z: -2.678345310936679e+10}, Velocity: math3d.Vec3{X: 5.872926054209884e+03, Y: 7.183114556123904e+03, Z: -3.585718628281587e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 4.431381727358100e+11, Y: 2.830268371248370e+12, Z: 4.774331040624430e+09}, Velocity: math3d.Vec3{X: -6.773197868232549e+03, Y: 7.454511765089536e+02, Z: 9.061926025469505e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.454140655478872e+12, Y: 2.456947873653358e+11, Z: -1.076939082717100e+11}, Velocity: math3d.Vec3{X: -3.473969241840845e+02, Y: 5.464286057522389e+03, Z: -1.045094350346878e+02}},
}

var golden1000 = []goldenBody{
	{Name: "Mercury", Position: math3d.Vec3{X: 3.100889058568648e+10, Y: 3.529016438905077e+10, Z: 3.668515101269854e+07}, Velocity: math3d.Vec3{X: -4.625950943765095e+04, Y: 3.419308060434307e+04, Z: 7.039148030577659e+03}},
	{Name: "Venus", Position: math3d.Vec3{X: 6.934565487516048e+10, Y: -8.380825203731258e+10, Z: -5.148989455775448e+09}, Velocity: math3d.Vec3{X: 2.675696465019307e+04, Y: 2.220711648013482e+04, Z: -1.241095875088886e+03}},
	{Name: "Earth", Position: math3d.Vec3{X: 9.242574605466663e+09, Y: -1.517819710437143e+11, Z: 3.759305552738705e+05}, Velocity: math3d.Vec3{X: 2.925220457391292e+04, Y: 1.700026420664466e+03, Z: 4.875408410613106e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -2.118384246702698e+11, Y: 1.307228755540598e+11, Z: 7.946415139178098e+09}, Velocity: math3d.Vec3{X: -1.181348079811894e+04, Y: -1.855499050055023e+04, Z: -9.835598962218742e+01}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.637076718274331e+11, Y: -2.873401037026093e+11, Z: 1.826906735832726e+10}, Velocity: math3d.Vec3{X: 4.445027416417601e+03, Y: -1.161850541794255e+04, Z: -5.142739505999990e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.142661557152073e+12, Y: -9.378194900579581e+11, Z: -2.908330542614966e+10}, Velocity: math3d.Vec3{X: 5.574287876756508e+03, Y: 7.438461537672410e+03, Z: -3.511508485941512e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 3.991969419050911e+11, Y: 2.834762662807522e+12, Z: 5.360954637173024e+09}, Velocity: math3d.Vec3{X: -6.788647051432859e+03, Y: 6.416121007294582e+02, Z: 9.043375066829860e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.451749653816699e+12, Y: 2.810952549437485e+11, Z: -1.083677399303817e+11}, Velocity: math3d.Vec3{X: -3.905641388789657e+02, Y: 5.461726829579288e+03, Z: -1.034621762163557e+02}},
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
