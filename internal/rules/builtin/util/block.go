package util

import "regexp"

var blockLookupRe = regexp.MustCompile(`^(LABEL|UUID|PARTLABEL|PARTUUID|ID)=.*`)

func IsBlockLookup(dev string) bool {
    return blockLookupRe.MatchString(dev)
}
