package session

import "testing"

func TestMachineLifecycle(t *testing.T) {
	m := NewMachine(3)
	for _, state := range []State{StateHandshaking, StateActive, StateMigrating, StateActive, StateDraining, StateClosed} {
		if err := m.Move(state); err != nil {
			t.Fatalf("Move(%s): %v", state, err)
		}
	}
	if m.State() != StateClosed || len(m.History()) != 3 {
		t.Fatalf("unexpected state/history: %s %#v", m.State(), m.History())
	}
	if err := m.Move(StateActive); err == nil {
		t.Fatal("closed session was revived")
	}
}

func TestMachineRejectsUnsafeSkip(t *testing.T) {
	m := NewMachine(4)
	if err := m.Move(StateActive); err == nil {
		t.Fatal("created session skipped authentication")
	}
}
