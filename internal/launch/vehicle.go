package launch

// Stage represents a single rocket stage.
type Stage struct {
	Name    string
	Isp     float64 // specific impulse (seconds)
	Thrust  float64 // thrust (Newtons)
	WetMass float64 // total mass with fuel (kg)
	DryMass float64 // mass without fuel (kg)
}

// Vehicle represents a launch vehicle with one or more stages.
type Vehicle struct {
	Name   string
	Stages []Stage
}

// Vehicles is the catalog of available vehicle presets.
var Vehicles = map[string]Vehicle{
	"generic": {
		Name: "Generic",
		Stages: []Stage{
			{Name: "Stage 1", Isp: 290, Thrust: 7e6, WetMass: 400000, DryMass: 30000},
			{Name: "Stage 2", Isp: 340, Thrust: 1e6, WetMass: 100000, DryMass: 6000},
		},
	},
	"falcon": {
		Name: "Falcon-like",
		Stages: []Stage{
			{Name: "Stage 1", Isp: 282, Thrust: 7.6e6, WetMass: 433100, DryMass: 25600},
			{Name: "Stage 2", Isp: 348, Thrust: 934000, WetMass: 111500, DryMass: 4000},
		},
	},
	"saturnv": {
		Name: "Saturn V-like",
		Stages: []Stage{
			{Name: "S-IC", Isp: 263, Thrust: 35.1e6, WetMass: 2290000, DryMass: 131000},
			{Name: "S-II", Isp: 421, Thrust: 5.141e6, WetMass: 496200, DryMass: 36200},
			{Name: "S-IVB", Isp: 421, Thrust: 1.033e6, WetMass: 123000, DryMass: 13300},
		},
	},
}

// GetVehicle returns a vehicle by key name. Returns the generic vehicle if not found.
func GetVehicle(name string) Vehicle {
	if v, ok := Vehicles[name]; ok {
		return v
	}
	return Vehicles["generic"]
}

// VehicleNames returns the available vehicle preset names in display order.
func VehicleNames() []string {
	return []string{"generic", "falcon", "saturnv"}
}
