package gll

import "testing"

const testMaxWeightKg = "MaxWeightKg"

func TestLimitTypeString(t *testing.T) {
	cases := []struct {
		value LimitType
		want  string
	}{
		{LimitTypeMaxCount, "MaxCount"},
		{LimitTypeMaxCountType, "MaxCountType"},
		{LimitTypeMaxWeightKg, testMaxWeightKg},
		{LimitTypeMaxTiltAngle, "MaxTiltAngle"},
		{LimitTypeMinTiltAngle, "MinTiltAngle"},
		{LimitTypeMinCount, "MinCount"},
		{LimitType(99), "Unknown"},
	}

	for _, tc := range cases {
		if got := tc.value.String(); got != tc.want {
			t.Errorf("LimitType(%d).String() = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestWarningTypeString(t *testing.T) {
	cases := []struct {
		value WarningType
		want  string
	}{
		{WarningTypeMaxCount, "MaxCount"},
		{WarningTypeMinCount, "MinCount"},
		{WarningTypeMaxWeightKg, testMaxWeightKg},
		{WarningTypeMaxTiltAngle, "MaxTiltAngle"},
		{WarningTypeMinTiltAngle, "MinTiltAngle"},
		{WarningType(99), "Unknown"},
	}

	for _, tc := range cases {
		if got := tc.value.String(); got != tc.want {
			t.Errorf("WarningType(%d).String() = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestLimitStruct(t *testing.T) {
	lim := Limit{
		Frame:      "frame1",
		Type:       LimitTypeMaxCount,
		BoxType:    "box1",
		LimitValue: 10.5,
	}

	if lim.Frame != "frame1" {
		t.Errorf("Limit.Frame = %q, want %q", lim.Frame, "frame1")
	}
	if lim.Type != LimitTypeMaxCount {
		t.Errorf("Limit.Type = %d, want %d", lim.Type, LimitTypeMaxCount)
	}
	if lim.LimitValue != 10.5 {
		t.Errorf("Limit.LimitValue = %f, want 10.5", lim.LimitValue)
	}
}

func TestWarningStruct(t *testing.T) {
	warn := Warning{
		Frame:      "frame1",
		Type:       WarningTypeMaxWeightKg,
		Text:       "Weight limit exceeded",
		LimitValue: 100.0,
	}

	if warn.Type.String() != testMaxWeightKg {
		t.Errorf("Warning.Type.String() = %q, want %s", warn.Type.String(), testMaxWeightKg)
	}
	if warn.Text != "Weight limit exceeded" {
		t.Errorf("Warning.Text = %q, want %q", warn.Text, "Weight limit exceeded")
	}
}

func TestFilterGroupStruct(t *testing.T) {
	grp := FilterGroup{
		Label:         "EQ Group 1",
		Key:           "eq1",
		IsOverridable: true,
		Filters: []FilterDefinition{
			{Label: "HPF", Key: "hpf1"},
			{Label: "LPF", Key: "lpf1"},
		},
	}

	if grp.Label != "EQ Group 1" {
		t.Errorf("FilterGroup.Label = %q, want %q", grp.Label, "EQ Group 1")
	}
	if !grp.IsOverridable {
		t.Error("FilterGroup.IsOverridable should be true")
	}
	if len(grp.Filters) != 2 {
		t.Errorf("len(FilterGroup.Filters) = %d, want 2", len(grp.Filters))
	}
	if grp.Filters[0].Label != "HPF" {
		t.Errorf("FilterGroup.Filters[0].Label = %q, want HPF", grp.Filters[0].Label)
	}
}

func TestDatabaseStruct(t *testing.T) {
	db := Database{
		SubVersion: 3,
		Limits: []Limit{
			{Type: LimitTypeMaxCount, LimitValue: 5},
		},
		Warnings: []Warning{
			{Type: WarningTypeMaxCount, LimitValue: 4},
		},
		FilterGroups: []FilterGroup{
			{Label: "Default", Key: "default"},
		},
	}

	if db.SubVersion != 3 {
		t.Errorf("Database.SubVersion = %d, want 3", db.SubVersion)
	}
	if len(db.Limits) != 1 {
		t.Errorf("len(Database.Limits) = %d, want 1", len(db.Limits))
	}
	if len(db.Warnings) != 1 {
		t.Errorf("len(Database.Warnings) = %d, want 1", len(db.Warnings))
	}
	if len(db.FilterGroups) != 1 {
		t.Errorf("len(Database.FilterGroups) = %d, want 1", len(db.FilterGroups))
	}
}

func TestDataFileStruct(t *testing.T) {
	df := DataFile{
		Key:      "image1",
		Filename: "front.png",
		Size:     1024,
		Offset:   1000,
	}

	if df.Key != "image1" {
		t.Errorf("DataFile.Key = %q, want image1", df.Key)
	}
	if df.Filename != "front.png" {
		t.Errorf("DataFile.Filename = %q, want front.png", df.Filename)
	}
	if df.Size != 1024 {
		t.Errorf("DataFile.Size = %d, want 1024", df.Size)
	}
}

func TestConnectorStruct(t *testing.T) {
	conn := Connector{
		UpperBox: "box_upper",
		LowerBox: "box_lower",
		Angles: []LabeledValueD{
			{Label: "Angle 1", Value: 15.0},
			{Label: "Angle 2", Value: 30.0},
		},
	}

	if conn.UpperBox != "box_upper" {
		t.Errorf("Connector.UpperBox = %q, want box_upper", conn.UpperBox)
	}
	if len(conn.Angles) != 2 {
		t.Errorf("len(Connector.Angles) = %d, want 2", len(conn.Angles))
	}
	if conn.Angles[0].Value != 15.0 {
		t.Errorf("Connector.Angles[0].Value = %f, want 15.0", conn.Angles[0].Value)
	}
}

func TestFrameStruct(t *testing.T) {
	frame := Frame{
		Label:        "Main Frame",
		Key:          "frame_main",
		Weight:       25.5,
		NextPivot:    &Vector3D{X: 100, Y: 0, Z: 50},
		CenterOfMass: &Vector3D{X: 50, Y: 0, Z: 25},
		PinPoints: []LabeledVector3D{
			{Label: "Pin 1", Vector: Vector3D{X: 0, Y: 0, Z: 0}},
		},
	}

	if frame.Label != "Main Frame" {
		t.Errorf("Frame.Label = %q, want Main Frame", frame.Label)
	}
	if frame.Weight != 25.5 {
		t.Errorf("Frame.Weight = %f, want 25.5", frame.Weight)
	}
	if frame.NextPivot.X != 100 {
		t.Errorf("Frame.NextPivot.X = %f, want 100", frame.NextPivot.X)
	}
	if len(frame.PinPoints) != 1 {
		t.Errorf("len(Frame.PinPoints) = %d, want 1", len(frame.PinPoints))
	}
}

func TestBoxTypeStruct(t *testing.T) {
	box := BoxType{
		Label:     "Main Speaker",
		Key:       "main",
		Weight:    35.0,
		NextPivot: &Vector3D{X: 500, Y: 400, Z: 300},
	}

	if box.Label != "Main Speaker" {
		t.Errorf("BoxType.Label = %q, want Main Speaker", box.Label)
	}
	if box.Weight != 35.0 {
		t.Errorf("BoxType.Weight = %f, want 35.0", box.Weight)
	}
	if box.NextPivot.X != 500 {
		t.Errorf("BoxType.NextPivot.X = %f, want 500", box.NextPivot.X)
	}
}

func TestGenSystemPresetStruct(t *testing.T) {
	preset := GenSystemPreset{
		Label: "Default Preset",
		Key:   "preset_default",
	}

	if preset.Label != "Default Preset" {
		t.Errorf("GenSystemPreset.Label = %q, want Default Preset", preset.Label)
	}
	if preset.Key != "preset_default" {
		t.Errorf("GenSystemPreset.Key = %q, want preset_default", preset.Key)
	}
}

func TestIncludeFileStruct(t *testing.T) {
	inc := IncludeFile{
		Label:    "Product Manual",
		Key:      "manual",
		Filename: "manual.pdf",
		Size:     512000,
	}

	if inc.Label != "Product Manual" {
		t.Errorf("IncludeFile.Label = %q, want Product Manual", inc.Label)
	}
	if inc.Filename != "manual.pdf" {
		t.Errorf("IncludeFile.Filename = %q, want manual.pdf", inc.Filename)
	}
}

func TestTransformerStruct(t *testing.T) {
	trans := Transformer{
		Label:         "Main Transformer",
		Key:           "trans1",
		LspkImpedance: 8.0,
		NetVoltage:    70.7,
		MaxPower:      100.0,
		TapSettings: []TapSetting{
			{Label: "Full", Key: "full", PowerRatio: 1.0},
			{Label: "Half", Key: "half", PowerRatio: 0.5},
		},
	}

	if trans.Label != "Main Transformer" {
		t.Errorf("Transformer.Label = %q, want Main Transformer", trans.Label)
	}
	if trans.LspkImpedance != 8.0 {
		t.Errorf("Transformer.LspkImpedance = %f, want 8.0", trans.LspkImpedance)
	}
	if len(trans.TapSettings) != 2 {
		t.Errorf("len(Transformer.TapSettings) = %d, want 2", len(trans.TapSettings))
	}
	if trans.TapSettings[1].PowerRatio != 0.5 {
		t.Errorf("Transformer.TapSettings[1].PowerRatio = %f, want 0.5", trans.TapSettings[1].PowerRatio)
	}
}

func TestClusterSetupItemStruct(t *testing.T) {
	item := ClusterSetupItem{
		Label: "Cluster 1",
		Key:   "cluster1",
		Setup: ClusterSetup{},
	}

	if item.Key != "cluster1" {
		t.Errorf("ClusterSetupItem.Key = %q, want cluster1", item.Key)
	}
}

func TestClusterSetupStruct(t *testing.T) {
	setup := ClusterSetup{
		Description: "Array Config",
		Boxes: []ClusterBox{
			{Label: "box1", BoxTypeKey: "main"},
		},
	}

	if setup.Description != "Array Config" {
		t.Errorf("ClusterSetup.Description = %q, want Array Config", setup.Description)
	}
	if len(setup.Boxes) != 1 {
		t.Errorf("len(ClusterSetup.Boxes) = %d, want 1", len(setup.Boxes))
	}
	if setup.Boxes[0].Label != "box1" {
		t.Errorf("ClusterSetup.Boxes[0].Label = %q, want box1", setup.Boxes[0].Label)
	}
}
