package domain

import "testing"

func TestValidName(t *testing.T) {
	// test cases taken from: https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html#bucket-names
	var tests = []struct {
		name  string
		input string
		valid bool
	}{
		{
			"forbidden prefix",
			"amzn-s3-demo-bucket1-a1b2c3d4-5678-90ab-cdef-example11111",
			false,
		},
		{
			"forbidden prefix",
			"amzn-s3-demo-bucket",
			false,
		},
		{
			"web domain",
			"example.com",
			true,
		},
		{
			"web domain",
			"www.example.com",
			true,
		},
		{
			"web domain",
			"my.example.s3.bucket",
			true,
		},
		{
			"contains underscores",
			"amzn_s3_demo_bucket",
			false,
		},
		{
			"contains uppercase letters",
			"AmznS3DemoBucket",
			false,
		},
		{
			"forbidden prefix and ends with hyphen",
			"amzn-s3-demo-bucket-",
			false,
		},
		{
			"contains two periods in a row",
			"example..com",
			false,
		},
		{
			"matches format of an IP address",
			"192.168.5.4",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := validName(test.input) == nil
			want := test.valid
			if got != want {
				t.Errorf("got valid: '%t', want valid: '%t'", got, want)
			}
		})
	}
}
