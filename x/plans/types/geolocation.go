package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// for convenience (calculate once only)
var (
	allGeoEnumRegionsList []int32
	allGeoEnumRegions     int32
)

// initialize convenience vars at start-up
func init() {
	for _, geoloc := range Geolocation_value {
		if geoloc != int32(Geolocation_GLS) && geoloc != int32(Geolocation_GL) {
			allGeoEnumRegionsList = append(allGeoEnumRegionsList, geoloc)
			allGeoEnumRegions |= geoloc
		}
	}
}

// IsValidGeoEnum tests the validity of a given geolocation
func IsValidGeoEnum(geoloc int32) bool {
	return geoloc != int32(Geolocation_GLS) && (geoloc & ^allGeoEnumRegions) == 0
}

// IsGeoEnumSingleBit returns true if at most one bit is set
func IsGeoEnumSingleBit(geoloc int32) bool {
	return (geoloc & (geoloc - 1)) == 0
}

// ParseGeoEnum parses a string into GeoEnum bitmask.
// The string may be a number or a comma-separated geolocations codes.
func ParseGeoEnum(arg string) (geoloc int32, err error) {
	geoloc64, err := strconv.ParseUint(arg, 10, 32)
	geoloc = int32(geoloc64)
	if err == nil {
		if geoloc != int32(Geolocation_GL) {
			if !IsValidGeoEnum(geoloc) {
				return 0, fmt.Errorf("invalid geolocation value: %s", arg)
			}
		}
		return geoloc, nil
	}

	split := strings.Split(arg, ",")
	for _, s := range split {
		val, ok := Geolocation_value[s]
		if !ok || val == int32(Geolocation_GLS) {
			return 0, fmt.Errorf("invalid geolocation code: %s", s)
		}
		geoloc |= val
	}

	return geoloc, nil
}

func GetAllGeolocations() []int32 {
	return allGeoEnumRegionsList
}

func GetGeolocationsFromUint(geoloc int32) []Geolocation {
	geoList := []Geolocation{}
	allGeos := GetAllGeolocations()
	for _, geo := range allGeos {
		if geo&geoloc != 0 {
			geoList = append(geoList, Geolocation(geo))
		}
	}

	return geoList
}

// allows unmarshaling parser func
func (g Geolocation) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(Geolocation_name[int32(g)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (g *Geolocation) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*g = Geolocation(Geolocation_value[j])
	return nil
}

type geoInfo struct {
	name string
	val  int32
}

func PrintGeolocations() string {
	var geos []geoInfo
	for geo, geoInt := range Geolocation_value {
		if geoInt == int32(Geolocation_GLS) {
			continue
		}
		geos = append(geos, geoInfo{name: geo, val: geoInt})
	}

	sort.Slice(geos, func(i, j int) bool {
		return geos[i].val < geos[j].val
	})

	var geosStr string
	for _, info := range geos {
		geosStr += info.String() + ", "
	}

	return geosStr[:len(geosStr)-2]
}

func (gi geoInfo) String() string {
	return gi.name + ": 0x" + strconv.FormatInt(int64(gi.val), 16)
}
