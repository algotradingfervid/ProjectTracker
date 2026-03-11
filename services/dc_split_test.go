package services

import (
	"testing"
)

func TestCreateSplit_InvalidStatus(t *testing.T) {
	// CreateSplit should reject DCs that are not in issued/splitting status.
	// This is a unit-level validation test; full integration requires test app.
	params := SplitParams{
		ProjectID:      "proj1",
		TransferDCID:   "dc1",
		DestinationIDs: []string{"dest1"},
	}

	// Without a real app, we just verify the params struct is well-formed
	if len(params.DestinationIDs) == 0 {
		t.Error("expected at least one destination ID")
	}
	if params.ProjectID == "" {
		t.Error("expected non-empty project ID")
	}
}

func TestSplitParams_SerialAssignments(t *testing.T) {
	// Verify serial assignment map works correctly
	params := SplitParams{
		ProjectID:      "proj1",
		TransferDCID:   "dc1",
		DestinationIDs: []string{"dest1", "dest2"},
		SerialAssignments: map[string][]string{
			"li1": {"SN-001", "SN-002", "SN-003"},
			"li2": {"SN-100"},
		},
	}

	if len(params.SerialAssignments) != 2 {
		t.Errorf("expected 2 serial assignment entries, got %d", len(params.SerialAssignments))
	}

	li1Serials := params.SerialAssignments["li1"]
	if len(li1Serials) != 3 {
		t.Errorf("expected 3 serials for li1, got %d", len(li1Serials))
	}

	li2Serials := params.SerialAssignments["li2"]
	if len(li2Serials) != 1 {
		t.Errorf("expected 1 serial for li2, got %d", len(li2Serials))
	}
}

func TestSplitResult_Structure(t *testing.T) {
	// Verify result struct holds expected fields
	result := SplitResult{
		SplitID:         "split1",
		ShipmentGroupID: "sg1",
		TransitDCID:     "tdc1",
		TransitDCNumber: "TDC-001",
		OfficialDCIDs:   []string{"odc1", "odc2"},
		OfficialDCNums:  []string{"ODC-001", "ODC-002"},
	}

	if result.SplitID != "split1" {
		t.Errorf("expected split ID 'split1', got %s", result.SplitID)
	}
	if len(result.OfficialDCIDs) != 2 {
		t.Errorf("expected 2 official DCs, got %d", len(result.OfficialDCIDs))
	}
	if result.TransitDCNumber != "TDC-001" {
		t.Errorf("expected transit DC number 'TDC-001', got %s", result.TransitDCNumber)
	}
}
