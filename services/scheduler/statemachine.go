package main

// transition is the outcome of feeding one check result into a monitor's
// state machine. Pure data: the prober applies it to the database.
type transition struct {
	NewState        string // unknown|up|down
	NewFails        int
	OpenIncident    bool // insert an incident (monitor just crossed its threshold)
	ResolveIncident bool // resolve the open incident (monitor recovered)
}

// evaluate advances the monitor state machine by one check result:
//
//	unknown → up   on first success
//	any     → fails+1 on failure; → down (+incident) at failure_threshold
//	down    → up (+resolve) on first success
//
// A monitor already down stays down on further failures without opening
// another incident (the open-incident unique index backs this up).
func evaluate(state string, consecutiveFails, failureThreshold int, ok bool) transition {
	if ok {
		return transition{
			NewState:        "up",
			NewFails:        0,
			ResolveIncident: state == "down",
		}
	}

	fails := consecutiveFails + 1
	if state != "down" && fails >= failureThreshold {
		return transition{NewState: "down", NewFails: fails, OpenIncident: true}
	}
	if state == "down" {
		return transition{NewState: "down", NewFails: fails}
	}
	// Below threshold: keep the current state (up stays up, unknown stays
	// unknown) while failures accumulate.
	return transition{NewState: state, NewFails: fails}
}
