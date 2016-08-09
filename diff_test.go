package api

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type innerStruct struct {
	A    string
	Data string
}

func (p innerStruct) ID() string {
	return string(p.A)
}

type myStruct struct {
	P   string
	F1  int        `diff:"value"`
	F2  int        `diff:"ignore"`
	F3  []myStruct `diff:"composition"`
	F4  []myStruct
	F5  []HasIdentifier
	F6  []int
	F7  []string
	F8  HasIdentifier
	F10 *myStruct
	F11 *innerStruct
	F12 innerStruct
}

func (p myStruct) ID() string {
	return string(p.P)
}

func diffReport(d *diff, report []string) []string {
	for k, v := range d.Param {
		line := fmt.Sprintf("%s.%s:%v->%v", d.ID, k, v.Current, v.Proposed)
		report = append(report, line)
	}

	for k, v := range d.Composition {
		if v.New != nil {
			for _, vv := range v.New {
				p := vv.(HasIdentifier)
				line := fmt.Sprintf("%s.%s:New=%s", d.ID, k, p.ID())
				report = append(report, line)
			}
		}
		if v.Deleted != nil {
			for _, vv := range v.Deleted {
				p := vv.(HasIdentifier)
				line := fmt.Sprintf("%s.%s:Deleted=%s", d.ID, k, p.ID())
				report = append(report, line)
			}
		}
		if v.Modified != nil {
			for _, dd := range v.Modified {
				line := fmt.Sprintf("%s.%s:Modified=%s", d.ID, k, dd.ID)
				report = append(report, line)
				report = diffReport(&dd, report)
			}
		}
	}

	return report
}

func TestCheckDiff2(t *testing.T) {

	testcase := []struct {
		name     string
		current  myStruct
		proposed myStruct
		report   []string
	}{
		{
			name:     "no_changes_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			report:   []string{},
		},
		{
			name:     "no_changes_f3f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:     "f1change_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 10, F2: 2, F3: []myStruct{}},
			report:   []string{"A.F1:1->10"},
		},
		{
			name:     "f1f2change",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 10, F2: 20, F3: []myStruct{}},
			report:   []string{"A.F1:1->10"},
		},
		{
			name:     "f3nilandnotnilchange",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:    "f3New_x2",
			current: myStruct{P: "A", F1: 1, F2: 2},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2}, {P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:New=B1", "A.F3:New=B2"},
		},
		{
			name: "f3NewAndDelete",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:Deleted=B1", "A.F3:New=B2"},
		},
		{
			name: "f3.B1_modified_notB200",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2}, {P: "B200", F1: 10, F2: 20},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 3, F2: 3}, {P: "B200", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:Modified=B1", "B1.F1:1->3"},
		},
		{
			name: "deep change",
			current: myStruct{P: "A", F3: []myStruct{
				{P: "B1", F3: []myStruct{
					{P: "B100", F12: innerStruct{A: "AA", Data: "d0"}},
				}},
			}},
			proposed: myStruct{P: "A", F3: []myStruct{
				{P: "B1", F3: []myStruct{
					{P: "B100", F12: innerStruct{A: "AA", Data: "d1"}},
				}},
			}},
			report: []string{"A.F3:Modified=B1", "AA.Data:d0->d1", "B1.F3:Modified=B100", "B100.F12:Modified=AA"},
		},
		{
			name: "doubleCompoAllTypeOfChange",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2, F3: []myStruct{
					{P: "C1", F1: 3, F2: 3}, {P: "C2", F1: 10, F2: 20},
				},
				},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 5, F2: 2, F3: []myStruct{
					{P: "C3", F1: 0, F2: 0}, {P: "C2", F1: 100, F2: 20},
				},
				},
			}},
			report: []string{"A.F3:Modified=B1", "B1.F1:1->5", "B1.F3:Deleted=C1", "B1.F3:Modified=C2", "B1.F3:New=C3", "C2.F1:10->100"},
		},
		{
			name:     "innerStructNoChange",
			current:  myStruct{P: "A", F11: &innerStruct{A: "AA"}},
			proposed: myStruct{P: "A", F11: &innerStruct{A: "AA"}},
			report:   []string{},
		},
		{
			name:     "innerStructNoChange",
			current:  myStruct{P: "A", F12: innerStruct{A: "AA"}},
			proposed: myStruct{P: "A", F12: innerStruct{A: "AA"}},
			report:   []string{},
		},
		{
			name:     "innerStructDataChange",
			current:  myStruct{P: "A", F12: innerStruct{A: "AA", Data: "oldValue"}},
			proposed: myStruct{P: "A", F12: innerStruct{A: "AA", Data: "newValue"}},
			report:   []string{"A.F12:Modified=AA", "AA.Data:oldValue->newValue"},
		},
		{
			name:     "innerStructReplaced",
			current:  myStruct{P: "A", F12: innerStruct{A: "AA"}},
			proposed: myStruct{P: "A", F12: innerStruct{A: "BB"}},
			report:   []string{"A.F12:Deleted=AA", "A.F12:New=BB"},
		},
		{
			name:     "innerStructNew",
			current:  myStruct{P: "A"},
			proposed: myStruct{P: "A", F11: &innerStruct{A: "AA"}},
			report:   []string{"A.F11:New=AA"},
		},
	}

	for _, test := range testcase {
		fmt.Printf("Test %s\n", test.name)
		d, err := checkDiff2(test.current, test.proposed)
		if err != nil {
			t.Errorf("Test %s failed with error %v", test.name, err)
			continue
		}
		report := []string{}
		report = diffReport(d, report)
		sort.Strings(report)
		sort.Strings(test.report)
		if !reflect.DeepEqual(report, test.report) {
			t.Errorf("Test %s did not give expected report:\nExpected:\n%v\nGot:\n%v\n", test.name, test.report, report)
			continue
		}
	}
}

func TestCheckDiff(t *testing.T) {

	testcase := []struct {
		name     string
		current  myStruct
		proposed myStruct
		report   []string
	}{
		{
			name:     "no_changes_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			report:   []string{},
		},
		{
			name:     "no_changes_f3f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:     "f1change_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 10, F2: 2, F3: []myStruct{}},
			report:   []string{"A.F1:1->10"},
		},
		{
			name:     "f1f2change",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 10, F2: 20, F3: []myStruct{}},
			report:   []string{"A.F1:1->10"},
		},
		{
			name:     "f3nilandnotnilchange",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:     "f3New_x2_f4nil",
			current:  myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{}},
			proposed: myStruct{P: "A", F1: 1, F2: 2},
			report:   []string{},
		},
		{
			name:    "f3New_x2",
			current: myStruct{P: "A", F1: 1, F2: 2},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2}, {P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:New=B1", "A.F3:New=B2"},
		},
		{
			name: "f3NewAndDelete",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:Deleted=B1", "A.F3:New=B2"},
		},
		{
			name: "f3.B1_modified",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2}, {P: "B2", F1: 10, F2: 20},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 3, F2: 3}, {P: "B2", F1: 10, F2: 20},
			}},
			report: []string{"A.F3:Modified=B1", "B1.F1:1->3"},
		},
		{
			name: "doubleCompoAllTypeOfChange",
			current: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 1, F2: 2, F3: []myStruct{
					{P: "C1", F1: 3, F2: 3}, {P: "C2", F1: 10, F2: 20},
				},
				},
			}},
			proposed: myStruct{P: "A", F1: 1, F2: 2, F3: []myStruct{
				{P: "B1", F1: 5, F2: 2, F3: []myStruct{
					{P: "C3", F1: 0, F2: 0}, {P: "C2", F1: 100, F2: 20},
				},
				},
			}},
			report: []string{"A.F3:Modified=B1", "B1.F1:1->5", "B1.F3:Deleted=C1", "B1.F3:Modified=C2", "B1.F3:New=C3", "C2.F1:10->100"},
		},
	}

	for _, test := range testcase {
		fmt.Printf("Test %s\n", test.name)
		d, err := checkDiff(test.current, test.proposed)
		if err != nil {
			t.Errorf("Test %s failed with error %v", test.name, err)
			continue
		}
		report := []string{}
		report = diffReport(d, report)
		sort.Strings(report)
		sort.Strings(test.report)
		if !reflect.DeepEqual(report, test.report) {
			t.Errorf("Test %s did not give expected report:\nExpected:\n%v\nGot:\n%v\n", test.name, test.report, report)
			continue
		}
	}
}

type PathI string

func (p PathI) Path() string {
	return string(p)
}

func TestCheckDiffInComposition(t *testing.T) {

	testcase := []struct {
		name            string
		current         interface{}
		proposed        interface{}
		expectedError   string
		expectedSame    []string
		expectedNew     []string
		expectedDeleted []string
	}{
		{
			name:            "Same",
			current:         []PathI{"SAME"},
			proposed:        []PathI{"SAME"},
			expectedError:   "",
			expectedSame:    []string{"SAME"},
			expectedNew:     []string{},
			expectedDeleted: []string{},
		},
		{
			name:            "OneNew",
			current:         []PathI{},
			proposed:        []PathI{"ANEW"},
			expectedError:   "",
			expectedSame:    []string{},
			expectedNew:     []string{"ANEW"},
			expectedDeleted: []string{},
		},
		{
			name:            "OneNewOneRemain",
			current:         []PathI{"Remain"},
			proposed:        []PathI{"Remain", "ANEW"},
			expectedError:   "",
			expectedSame:    []string{"Remain"},
			expectedNew:     []string{"ANEW"},
			expectedDeleted: []string{},
		},
		{
			name:            "OneDel",
			current:         []PathI{"Del"},
			proposed:        []PathI{},
			expectedError:   "",
			expectedSame:    []string{},
			expectedNew:     []string{},
			expectedDeleted: []string{"Del"},
		},
		{
			name:            "Mix",
			current:         []PathI{"Del", "cur"},
			proposed:        []PathI{"cur", "New"},
			expectedError:   "",
			expectedSame:    []string{"cur"},
			expectedNew:     []string{"New"},
			expectedDeleted: []string{"Del"},
		},
		{
			name:            "errorType",
			current:         []string{"toto"},
			proposed:        []string{},
			expectedError:   "Compisition of non-PathIdentifier in current: string",
			expectedSame:    []string{},
			expectedNew:     []string{},
			expectedDeleted: []string{},
		},
	}

	for _, test := range testcase {

		s, n, d, e := checkDiffInComposition(test.current, test.proposed)
		if e != nil && test.expectedError == "" {
			t.Errorf("Test %s, unexpected error:%v", test.name, e)
			continue
		}

		if test.expectedError != "" && e == nil {
			t.Errorf("Test %s, go not error but was expecting error:%s", test.name, test.expectedError)
			continue
		}

		if test.expectedError != "" && test.expectedError != e.Error() {
			t.Errorf("Test %s, bad Error.\nExpected: %s\nGot:%s", test.name, test.expectedError, e.Error())
			continue
		}

		if len(s) != len(test.expectedSame) {
			t.Errorf("Test %s, Same item, len are different, expected %d, got %d, values:\nExpected:%v\n,Got:%v", test.name, len(test.expectedSame), len(s), test.expectedSame, s)
		} else {
			for i := 0; i < len(s); i++ {
				if s[i][0].ID() != test.expectedSame[i] {
					t.Errorf("Test %s, Incorrect Path in Same at index %d\nExpected:%v\nGot:%v", test.name, i, test.expectedSame, s)
				}
			}
		}

		if len(n) != len(test.expectedNew) {
			t.Errorf("Test %s, New item, len are different, expected %d, got %d, values:\nExpected:%v\n,Got:%v", test.name, len(test.expectedNew), len(n), test.expectedNew, n)
		} else {
			for i := 0; i < len(n); i++ {
				if n[i].ID() != test.expectedNew[i] {
					t.Errorf("Test %s, Incorrect Path in New at index %d\nExpected:%v\nGot:%v", test.name, i, test.expectedNew, n)
				}
			}
		}

		if len(d) != len(test.expectedDeleted) {
			t.Errorf("Test %s, Deleted item, len are different, expected %d, got %d, values:\nExpected:%v\n,Got:%v", test.name, len(test.expectedDeleted), len(d), test.expectedDeleted, d)
		} else {
			for i := 0; i < len(d); i++ {
				if d[i].ID() != test.expectedDeleted[i] {
					t.Errorf("Test %s, Incorrect Path in Deleted at index %d\nExpected:%v\nGot:%v", test.name, i, test.expectedDeleted, d)
				}
			}
		}

	}

}

type testStruct struct{}

func (t *testStruct) ID() string {
	return "0"
}

type testStructV struct{}

func (t testStructV) ID() string {
	return "V"
}

func TestIdentifierFormInterface(t *testing.T) {

	var iNil HasIdentifier
	iNil = (*testStruct)(nil)

	testcase := []struct {
		name           string
		input          interface{}
		expectedErrStr string
		expectedID     string
	}{
		{
			name:           "not init interface",
			expectedErrStr: "nil interface cannot get identifier",
		},
		{
			name:           "bad interface type",
			input:          new(string),
			expectedErrStr: "type assertion to 'hasIdentifier' failed",
		},
		{
			name:           "nil value",
			input:          iNil,
			expectedErrStr: "nil pointed value",
		},
		{
			name:       "valid",
			input:      &testStruct{},
			expectedID: "0",
		},
		{
			name:       "valid V",
			input:      testStructV{},
			expectedID: "V",
		},
	}

	for _, tc := range testcase {
		//fmt.Printf("Test %s:\n", tc.name)
		i, e := identifierFormInterface(tc.input)
		if len(tc.expectedErrStr) > 0 && (e == nil) {
			t.Errorf("%s : no error returned, but expect error: %s", tc.name, tc.expectedErrStr)
			continue
		}
		if len(tc.expectedErrStr) > 0 && tc.expectedErrStr != e.Error() {
			t.Errorf("%s : \nexpectedError:\n%s\nGot:\n%s\n", tc.name, tc.expectedErrStr, e.Error())
			continue
		}
		if len(tc.expectedErrStr) == 0 && e != nil {
			t.Errorf("%s : \nexpectedError:\n%s\nGot:\n%s\n", tc.name, tc.expectedErrStr, e.Error())
			continue
		}
		if len(tc.expectedErrStr) > 0 && tc.expectedErrStr == e.Error() {
			continue // ok error case match
		}
		if i.ID() != tc.expectedID {
			t.Errorf("%s : Bad Id\nExpected:%s  Got:%s\n", tc.name, tc.expectedID, i.ID())
			continue
		}

	}
}
