package webclient_test

import "testing"

func TestSomeFunctionOrMethod(t *testing.T) {
	testCases := []struct {
		desc     string
		term1    int
		term2    int
		sum      int
		expected bool
	}{
		{
			desc:     "verify that 1 + 1 = 2",
			term1:    1,
			term2:    1,
			sum:      2,
			expected: true,
		},
		{
			desc:     "verify that 1 + 2 != 2",
			term1:    1,
			term2:    2,
			sum:      2,
			expected: false,
		},
	}
	for _, c := range testCases {
		t.Run(c.desc, func(t *testing.T) {
			if (c.term1+c.term2 == c.sum) != c.expected {
				t.Fatalf("expected %d + %d = %d to be %v", c.term1, c.term2, c.sum, c.expected)
			}
		})
	}
}

func TestAnotherFunctionOrMethod(t *testing.T) {
	testCases := []struct {
		desc string
	}{
		{
			desc: "",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

		})
	}
}
