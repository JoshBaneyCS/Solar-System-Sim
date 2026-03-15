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
	{Name: "Mercury", Position: math3d.Vec3{X: -2.998050661944995e+10, Y: 3.837189492086417e+10, Z: 5.886361551274197e+09}, Velocity: math3d.Vec3{X: -4.823982841295216e+04, Y: -2.807767571051709e+04, Z: 2.134089252815371e+03}},
	{Name: "Venus", Position: math3d.Vec3{X: -1.059535261813235e+11, Y: -1.882757140603617e+10, Z: 5.859097985072027e+09}, Velocity: math3d.Vec3{X: 5.905421445407925e+03, Y: -3.465063655310046e+04, Z: -8.145128728484832e+02}},
	{Name: "Earth", Position: math3d.Vec3{X: -1.398307919832044e+11, Y: -5.405512215904287e+10, Z: 5.391508244695966e+03}, Velocity: math3d.Vec3{X: 1.025746908013374e+04, Y: -2.790031104814961e+04, Z: 1.545935318722349e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -1.022683467478876e+11, Y: 2.204221664159921e+11, Z: 7.132305515695683e+09}, Velocity: math3d.Vec3{X: -2.106041994569434e+04, Y: -8.139496642554389e+03, Z: 3.471718880411328e+02}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.885520918321320e+11, Y: -2.107096405710771e+11, Z: 1.850821009815355e+10}, Velocity: math3d.Vec3{X: 3.216642381359221e+03, Y: -1.201261145666886e+04, Z: -2.232351896609182e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.105566423231063e+12, Y: -9.851994769593331e+11, Z: -2.678345310936499e+10}, Velocity: math3d.Vec3{X: 5.872926054279810e+03, Y: 7.183114556018623e+03, Z: -3.585718628231376e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 4.431381727358097e+11, Y: 2.830268371248371e+12, Z: 4.774331040625253e+09}, Velocity: math3d.Vec3{X: -6.773197868235236e+03, Y: 7.454511765154811e+02, Z: 9.061926025697599e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.454140655478871e+12, Y: 2.456947873653347e+11, Z: -1.076939082717095e+11}, Velocity: math3d.Vec3{X: -3.473969241876418e+02, Y: 5.464286057518965e+03, Z: -1.045094350333501e+02}},
	{Name: "Pluto", Position: math3d.Vec3{X: -3.012084005221812e+12, Y: -3.030084869871788e+12, Z: 1.196923700919022e+12}, Velocity: math3d.Vec3{X: 4.156517610417645e+03, Y: -4.421969206817446e+03, Z: -7.300683604220335e+02}},
}

var golden1000 = []goldenBody{
	{Name: "Mercury", Position: math3d.Vec3{X: 3.100889058745589e+10, Y: 3.529016438760774e+10, Z: 3.668515073653610e+07}, Velocity: math3d.Vec3{X: -4.625950943638026e+04, Y: 3.419308060595286e+04, Z: 7.039148030568767e+03}},
	{Name: "Venus", Position: math3d.Vec3{X: 6.934564877366739e+10, Y: -8.380826706131313e+10, Z: -5.148989308761284e+09}, Velocity: math3d.Vec3{X: 2.675696639654752e+04, Y: 2.220711056091652e+04, Z: -1.241096056798710e+03}},
	{Name: "Earth", Position: math3d.Vec3{X: 9.242571074567766e+09, Y: -1.517819751426132e+11, Z: 3.759308580520449e+05}, Velocity: math3d.Vec3{X: 2.925220394439241e+04, Y: 1.700024748589233e+03, Z: 4.875415409659395e-02}},
	{Name: "Mars", Position: math3d.Vec3{X: -2.118384253708217e+11, Y: 1.307228764499832e+11, Z: 7.946415175455531e+09}, Velocity: math3d.Vec3{X: -1.181348103084074e+04, Y: -1.855499026387394e+04, Z: -9.835597886499093e+01}},
	{Name: "Jupiter", Position: math3d.Vec3{X: -7.637076718557493e+11, Y: -2.873401037117965e+11, Z: 1.826906735946800e+10}, Velocity: math3d.Vec3{X: 4.445027408573593e+03, Y: -1.161850542063129e+04, Z: -5.142739474251705e+01}},
	{Name: "Saturn", Position: math3d.Vec3{X: 1.142661557154642e+12, Y: -9.378194900617136e+11, Z: -2.908330542597362e+10}, Velocity: math3d.Vec3{X: 5.574287877479378e+03, Y: 7.438461536634635e+03, Z: -3.511508485460465e+02}},
	{Name: "Uranus", Position: math3d.Vec3{X: 3.991969419049934e+11, Y: 2.834762662807740e+12, Z: 5.360954637255160e+09}, Velocity: math3d.Vec3{X: -6.788647051460449e+03, Y: 6.416121007946948e+02, Z: 9.043375069111416e+01}},
	{Name: "Neptune", Position: math3d.Vec3{X: 4.451749653816566e+12, Y: 2.810952549436245e+11, Z: -1.083677399303338e+11}, Velocity: math3d.Vec3{X: -3.905641389144059e+02, Y: 5.461726829544816e+03, Z: -1.034621762030020e+02}},
	{Name: "Pluto", Position: math3d.Vec3{X: -2.985053803975452e+12, Y: -3.058642083850327e+12, Z: 1.192154648470679e+12}, Velocity: math3d.Vec3{X: 4.186092719549693e+03, Y: -4.391938891406368e+03, Z: -7.418535912183086e+02}},
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
