// +build gofuzz

package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2019 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

func FuzzV1(data []byte) int {
	_, err := parseV1Procfile(data, &Config{})

	if err != nil {
		return 1
	}

	return 0
}

func FuzzV2(data []byte) int {
	_, err := parseV2Procfile(data, &Config{})

	if err != nil {
		return 1
	}

	return 0
}
