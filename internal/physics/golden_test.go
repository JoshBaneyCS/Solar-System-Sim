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
	{Name: "Mercury", Position: math3d.Vec3{X: -2.998050661944295e+10, Y: 3.837189492087231e+10, Z: 5.886361551271410e+09}, Velocity: math3d.Vec3{X: -4.823982841293426e+04, Y: -2.807767571049468e+04, Z: 2.134089252808137e+03}},
	{Name: "Venus", Position: math3d.Vec3{X: -1.059535261813155e+11, Y: -1.882757140602798e+10, Z: 5.859097985068799e+09}, Velocity: math3d.Vec3{X: 5.905421445430373e+03, Y: -3.465063655307773e+04, Z: -8.145128728574438e+02}},
	{Name: "Earth", Position: math3d.Vec3{X: -1.398307919831964e+11, Y: -5.405512215903449e+10, Z: 5.391504884953621e+03}, Velocity: math3d.Vec3{X: 1.025746908015638e+04, Y: -2.790031104812627e+04, Z: 1.545934386008378e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -1.022683467478806e+11, Y: 2.204221664160000e+11, Z: 7.132305515692800e+09}, Velocity: math3d.Vec3{X: -2.106041994567470e+04, Y: -8.139496642532460e+03, Z: 3.471718880331171e+02}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.885520918321219e+11, Y: -2.107096405710655e+11, Z: 1.850821009814864e+10}, Velocity: math3d.Vec3{X: 3.216642381384977e+03, Y: -1.201261145663619e+04, Z: -2.232351897974074e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.105566423231072e+12, Y: -9.851994769593292e+11, Z: -2.678345310936755e+10}, Velocity: math3d.Vec3{X: 5.872926054303792e+03, Y: 7.183114556030508e+03, Z: -3.585718628302642e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 4.431381727358116e+11, Y: 2.830268371248374e+12, Z: 4.774331040624437e+09}, Velocity: math3d.Vec3{X: -6.773197868228657e+03, Y: 7.454511765266139e+02, Z: 9.061926025471018e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.454140655478873e+12, Y: 2.456947873653359e+11, Z: -1.076939082717101e+11}, Velocity: math3d.Vec3{X: -3.473969241793356e+02, Y: 5.464286057522603e+03, Z: -1.045094350348015e+02}},
}

var golden1000 = []goldenBody{
	{Name: "Mercury", Position: math3d.Vec3{X: 3.100889058540062e+10, Y: 3.529016438931814e+10, Z: 3.668515105087548e+07}, Velocity: math3d.Vec3{X: -4.625950943782856e+04, Y: 3.419308060412569e+04, Z: 7.039148030589879e+03}},
	{Name: "Venus", Position: math3d.Vec3{X: 6.934564877439389e+10, Y: -8.380826705956564e+10, Z: -5.148989308961560e+09}, Velocity: math3d.Vec3{X: 2.675696639647556e+04, Y: 2.220711056156671e+04, Z: -1.241096056810779e+03}},
	{Name: "Earth", Position: math3d.Vec3{X: 9.242571075515837e+09, Y: -1.517819751414212e+11, Z: 3.759305716259071e+05}, Velocity: math3d.Vec3{X: 2.925220394462762e+04, Y: 1.700024749031426e+03, Z: 4.875408832521504e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -2.118384253701387e+11, Y: 1.307228764507687e+11, Z: 7.946415175170432e+09}, Velocity: math3d.Vec3{X: -1.181348103065369e+04, Y: -1.855499026366284e+04, Z: -9.835597894264092e+01}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.637076718548099e+11, Y: -2.873401037106146e+11, Z: 1.826906735897265e+10}, Velocity: math3d.Vec3{X: 4.445027408836585e+03, Y: -1.161850542030207e+04, Z: -5.142739488068629e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.142661557155498e+12, Y: -9.378194900612838e+11, Z: -2.908330542622787e+10}, Velocity: math3d.Vec3{X: 5.574287877716351e+03, Y: 7.438461536753979e+03, Z: -3.511508486163058e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 3.991969419052292e+11, Y: 2.834762662808156e+12, Z: 5.360954637173619e+09}, Velocity: math3d.Vec3{X: -6.788647051395264e+03, Y: 6.416121009065341e+02, Z: 9.043375066846968e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.451749653816847e+12, Y: 2.810952549437563e+11, Z: -1.083677399303855e+11}, Velocity: math3d.Vec3{X: -3.905641388314800e+02, Y: 5.461726829581597e+03, Z: -1.034621762174968e+02}},
}

func testGoldenBaseline(t *testing.T, steps int, expected []goldenBody) {
	t.Helper()
	sim := NewSimulator()
	sim.Integrator = IntegratorRK4 // golden values computed with RK4
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
	sim.Integrator = IntegratorRK4 // golden values computed with RK4
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
