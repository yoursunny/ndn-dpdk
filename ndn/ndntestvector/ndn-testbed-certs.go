package ndntestvector

import (
	"encoding/base64"
	"strings"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// NDN testbed certificates.
// Each is a function that returns the certificate Data packet.
var (
	TestbedArizona20200301 = makeDataFromBase64(`
		Bv0CxgcxCANuZG4IA2VkdQgHYXJpem9uYQgDS0VZCAgk8wnxnkmjlwgCTkEICf0A
		AAFwnGPguhQJGAECGQQANu6AFf0BJjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
		AQoCggEBAMM/l8Stuasx/HUfl4B2yzFGHpWFsEriuaAuH/getpgfE7xLvQ+jWljS
		P0WC5p8dERE+m4/hTSrw09XveXJ+9xhSIVMW0bGc9sFVbGV3qMBtifqqGYUGgv65
		u8Elj/B+aYrAN6KO4LX0f7S1K9E7iSwRWDxbTvuDDHDeiyoxJi7pmcv6EzQSlD4i
		vhdSQKbZv1Sz7iuIL57dmJeB6eMA3ttHvU/YCrSD46hghYbh9VNaZESBVfutwVlJ
		1tfVj5/LmmzrEdP067I6aaBMUT2TJ7VdDtw4PFwJVb74XUL6flHzr4V/QI4cUyQn
		o3rAJ9+95eU2VsPwhutunjK/XCl1eJ0CAwEAARb9ARAbAQMcFgcUCANuZG4IA0tF
		WQgIZZ1/pcWBEH39AP0m/QD+DzIwMjAwMzAxVDE3NTU1Nv0A/w8yMDIxMDMwMlQx
		NzU1NTb9AQLH/QIAD/0CAQdhZHZpc29y/QICAP0CADf9AgEFZW1haWz9AgIqL25k
		bi9lZHUvYXJpem9uYS9Ab3BlcmF0b3JzLm5hbWVkLWRhdGEubmV0/QIAKf0CAQhm
		dWxsbmFtZf0CAhlUaGUgVW5pdmVyc2l0eSBvZiBBcml6b25h/QIADf0CAQVncm91
		cP0CAgD9AgAP/QIBB2hvbWV1cmz9AgIA/QIAJP0CAQxvcmdhbml6YXRpb279AgIQ
		TkROIFRlc3RiZWQgUm9vdBdIMEYCIQCRlyhpTVvQaBSOJOccmnRRJ5+xGQFi1BeN
		53zDaGdfGgIhAL/jklfHm+e1Rj2FxlaW0sSuEAJmYbq6dIKD7GgkNhhy`)

	TestbedShijunxiao20200301 = makeDataFromBase64(`
		Bv0CuwdBCANuZG4IA2VkdQgHYXJpem9uYQgCY3MICnNoaWp1bnhpYW8IA0tFWQgI
		Ixof9YCxm6EIAk5BCAn9AAABcJ0dzE8UCRgBAhkEADbugBVbMFkwEwYHKoZIzj0C
		AQYIKoZIzj0DAQcDQgAEQpx8nAasqj7OX8LEGeCV3RkEs4U74ArvX3OsDyDy3/nZ
		bfyqZe7ExWqQ6tmuRz8i0FnZzIGZWOK05RFQ/1UlOhb9AQgbAQEcJAciCANuZG4I
		A2VkdQgHYXJpem9uYQgDS0VZCAgk8wnxnkmjl/0A/Sb9AP4PMjAyMDAzMDFUMjEx
		OTAw/QD/DzIwMjEwMzAyVDIxMTkwMP0BArH9AgAP/QIBB2Fkdmlzb3L9AgIA/QIA
		Jv0CAQVlbWFpbP0CAhlzaGlqdW54aWFvQGNzLmFyaXpvbmEuZWR1/QIAG/0CAQhm
		dWxsbmFtZf0CAgtKdW54aWFvIFNoaf0CAA39AgEFZ3JvdXD9AgIA/QIAD/0CAQdo
		b21ldXJs/QICAP0CAC39AgEMb3JnYW5pemF0aW9u/QICGVRoZSBVbml2ZXJzaXR5
		IG9mIEFyaXpvbmEX/QEALoqLwjsB3krtNZmUzF5+0rJ+7vRkDhLssTivzrNt4x49
		V70Ktb9hFLYaSJ0/zghUlKSwvEcw+A8efnCbA4YL1RRRoTV4e9ptvURvglEalTPg
		njUVbMgvrkTClnJVQ4spd37cVu4/oXytQ+Zuzs46heGuf8crExYF+W6q5JL+G6lK
		DuGC/AonspPbBZlDQuCWfmvOhxduGTj9RPFFHcWwotsYwSj78gNgvrr9+SjNNsvB
		6k1UQlj4qIaX7KKl5u1E5ugrAVMkbShxMbguCnhxjzJbpyIgO+VyBKaV7h93c+CF
		FfTyYyzhIxl1J9EKcy6SOpvNDekotKOJ58SlGiI2wQ==`)

	TestbedRootX3 = makeDataFromBase64(`
		Bv0BSQckCANuZG4IA0tFWQgI7PFMjlEjFeAIA25kbggJ/QAAAXXmfzIQFAkYAQIZBAA27oAVWzBZ
		MBMGByqGSM49AgEGCCqGSM49AwEHA0IABBsft2OBb2KNXknCL4A++JUIUHczeM6tNtXaKfLe5Bnx
		KXxnSn9hxqZ5+P6qBfYidclGRP+zWvM8zuMU+kaSDNEWcBsBAxwWBxQIA25kbggDS0VZCAjs8UyO
		USMV4P0A/Sb9AP4PMjAyMDExMjBUMTYzMTM3/QD/DzIwMjQxMjMxVDIzNTk1Of0BAif9AgAj/QIB
		CGZ1bGxuYW1l/QICE05ETiBUZXN0YmVkIFJvb3QgWDMXRzBFAiEA/Ia7U+qGL01yLaX8uDSINwKw
		eLdnUIYCnIXms6goCtoCIFPAsXZhQXYOZZa6HkBxLZz2tqh3DqiLkZoY4lDYCcWp`)

	TestbedNeu20201217 = makeDataFromBase64(`
		Bv0B7QctCANuZG4IA2VkdQgDbmV1CANLRVkICBVH+ZVHoW16CAJOQQgJ/QAAAXZ2
		tj9+FAkYAQIZBAA27oAVWzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABFNOfYoi
		Vphi6wFSI4HTjoUELYyqrNMPivUxCGQho4ifiWIzJdBj7PklvDP4/wL/aj5asHP3
		Tbjy6uiJ0EAi3xMW/QEKGwEDHBYHFAgDbmRuCANLRVkICOzxTI5RIxXg/QD9Jv0A
		/g8yMDIwMTIxN1QxNjM3MDP9AP8PMjAyMTEyMThUMTYzNzAz/QECwf0CAA/9AgEH
		YWR2aXNvcv0CAgD9AgAz/QIBBWVtYWls/QICJi9uZG4vZWR1L25ldS9Ab3BlcmF0
		b3JzLm5hbWVkLWRhdGEubmV0/QIAJ/0CAQhmdWxsbmFtZf0CAhdOb3J0aGVhc3Rl
		cm4gVW5pdmVyc2l0ef0CAA39AgEFZ3JvdXD9AgIA/QIAD/0CAQdob21ldXJs/QIC
		AP0CACT9AgEMb3JnYW5pemF0aW9u/QICEE5ETiBUZXN0YmVkIFJvb3QXRjBEAiB0
		uX44O3vhTLx3ioGAIWsXl5Htbskry+hcczZ6zVY+NgIgT4nBoHDJkbeeAg7+tpAY
		E2rs8ThtmNI1je+CP5CfKOE=`)
)

func makeDataFromBase64(input string) func() ndn.Data {
	return func() ndn.Data {
		input = strings.NewReplacer("\n", "", "\t", "").Replace(input)
		wire, e := base64.StdEncoding.DecodeString(input)
		if e != nil {
			panic(e)
		}
		var pkt ndn.Packet
		e = tlv.Decode(wire, &pkt)
		if e != nil {
			panic(e)
		}
		return *pkt.Data
	}
}
