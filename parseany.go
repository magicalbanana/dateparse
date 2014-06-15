package dateparse

import (
	"fmt"
	u "github.com/araddon/gou"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

type DateState int

const (
	ST_START DateState = iota
	ST_DIGIT
	ST_DIGITDASH
	ST_DIGITDASHWS
	ST_DIGITDASHT
	ST_DIGITCOMMA
	ST_DIGITCOLON
	ST_DIGITSLASH
	ST_DIGITSLASHWS
	ST_DIGITSLASHWSCOLON
	ST_DIGITSLASHWSCOLONCOLON
	ST_ALPHA
	ST_ALPHAWS
	ST_ALPHAWSCOMMA
	ST_ALPHAWSALPHA
	ST_ALPHACOMMA
	ST_ALPHACOMMADASH
	ST_ALPHACOMMADASHDASH
	//ST_ALPHADIGIT
)

var _ = u.EMPTY

// Given an unknown date format, detect the type, parse, return time
func ParseAny(datestr string) (time.Time, error) {

	state := ST_START

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for i := 0; i < len(datestr); i++ {
		r, bytesConsumed := utf8.DecodeRuneInString(datestr[i:])
		if bytesConsumed > 1 {
			i += (bytesConsumed - 1)
		}

		switch state {
		case ST_START:
			if unicode.IsDigit(r) {
				state = ST_DIGIT
			} else if unicode.IsLetter(r) {
				state = ST_ALPHA
			}
		case ST_DIGIT: // starts digits
			if unicode.IsDigit(r) {
				continue
			}
			switch r {
			case ',':
				state = ST_DIGITCOMMA
			case '-':
				state = ST_DIGITDASH
			case ':':
				state = ST_DIGITCOLON
			case '/':
				state = ST_DIGITSLASH
			}
		case ST_DIGITDASH: // starts digit then dash 02-
			// 2006-01-02T15:04:05Z07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2012-08-03 18:31:59.257000000
			// 2014-04-26 17:24:37.3186369
			// 2016-03-14 00:00:00.000
			// 2014-05-11 08:20:13,787
			// 2006-01-02
			// 2013-04-01 22:43:22
			// 2014-04-26 05:24:37 PM
			switch {
			case r == ' ':
				state = ST_DIGITDASHWS
			case r == 'T':
				state = ST_DIGITDASHT
			}
		case ST_DIGITDASHWS: // starts digit then dash 02- then whitespace
			// 2014-04-26 17:24:37.3186369
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			// 2013-04-01 22:43:22
			// 2014-05-11 08:20:13,787
			// 2014-04-26 05:24:37 PM
			switch r {
			case 'A', 'P':
				if len(datestr) == len("2014-04-26 03:24:37 PM") {
					if t, err := time.Parse("2006-01-02 03:04:05 PM", datestr); err == nil {
						return t, nil
					} else {
						//u.Error(err)
					}
				}
			case ',':
				if len(datestr) == len("2014-05-11 08:20:13,787") {
					// go doesn't seem to parse this one natively?   or did i miss it?
					if t, err := time.Parse("2006-01-02 03:04:05", datestr[:i]); err == nil {
						ms, err := strconv.Atoi(datestr[i+1:])
						if err == nil {
							return time.Unix(0, t.UnixNano()+int64(ms)*1e6), nil
						}
						//u.Error(err)
					} else {
						//u.Error(err)
					}
				}
			}
		case ST_DIGITDASHT: // starts digit then dash 02-  then T
			// 2006-01-02T15:04:05Z07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			if len(datestr) == len("2006-01-02T15:04:05Z07:00") {
				if t, err := time.Parse("2006-01-02T15:04:05Z07:00", datestr); err == nil {
					return t, nil
				} else {
					//u.Error(err)
				}
			} else if len(datestr) == len("2006-01-02T15:04:05.999999999Z07:00") {
				if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", datestr); err == nil {
					return t, nil
				} else {
					//u.Error(err)
				}
			}
		case ST_DIGITSLASH: // starts digit then slash 02/
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			if unicode.IsDigit(r) || r == '/' {
				continue
			}
			switch r {
			case ' ':
				state = ST_DIGITSLASHWS
			}
		case ST_DIGITSLASHWS: // starts digit then slash 02/ more digits/slashes then whitespace
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = ST_DIGITSLASHWSCOLON
			}
		case ST_DIGITSLASHWSCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = ST_DIGITSLASHWSCOLONCOLON
			}
		case ST_ALPHA: // starts alpha
			// May 8, 2009 5:57:51 PM
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Monday, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 -0700
			if unicode.IsLetter(r) {
				continue
			}
			switch {
			case r == ' ':
				state = ST_ALPHAWS
			case r == ',':
				state = ST_ALPHACOMMA
				// case unicode.IsDigit(r):
				// 	state = ST_ALPHADIGIT
			}
		case ST_ALPHAWS: // Starts alpha then whitespace
			switch {
			// case r == ' ':
			// 	state = ST_ALPHAWSWS
			case r == ',':
				state = ST_ALPHAWSCOMMA
			case unicode.IsLetter(r):
				state = ST_ALPHAWSALPHA
			}
		case ST_ALPHACOMMA: // Starts alpha then comma
			// Mon, 02 Jan 2006 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 -0700
			// Monday, 02-Jan-06 15:04:05 MST
			switch {
			case r == '-':
				state = ST_ALPHACOMMADASH
			}

		case ST_ALPHACOMMADASH: // Starts alpha then comma and one dash
			// Mon, 02 Jan 2006 15:04:05 -0700
			// Monday, 02-Jan-06 15:04:05 MST
			switch {
			case r == '-':
				state = ST_ALPHACOMMADASHDASH
			}
		case ST_ALPHAWSCOMMA: // Starts Alpha, whitespace, digit, comma
			// May 8, 2009 5:57:51 PM
			if t, err := time.Parse("Jan 2, 2006 3:04:05 PM", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		case ST_ALPHAWSALPHA: // Starts Alpha, whitespace, alpha
			// ANSIC       = "Mon Jan _2 15:04:05 2006"
			// UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
			// RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
			if len(datestr) == len("Mon Jan _2 15:04:05 2006") {
				if t, err := time.Parse(time.ANSIC, datestr); err == nil {
					return t, nil
				} else {
					//u.Errorf("'%s'  err=%v", datestr, err)
					break iterRunes
				}
			} else if len(datestr) == len("Mon Jan _2 15:04:05 MST 2006") {
				if t, err := time.Parse(time.UnixDate, datestr); err == nil {
					return t, nil
				} else {
					//u.Error(err)
					break iterRunes
				}
			} else if len(datestr) == len("Mon Jan 02 15:04:05 -0700 2006") {
				if t, err := time.Parse(time.RubyDate, datestr); err == nil {
					return t, nil
				} else {
					//u.Error(err)
					break iterRunes
				}
			}
		default:
			//u.Infof("no case for: %d", state)
			break iterRunes
		}
	}

	switch state {
	case ST_DIGIT:
		// unixy timestamps ish
		if len(datestr) >= len("13980450781991351") {
			if nanoSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, nanoSecs), nil
			} else {
				//u.Error(err)
			}
		} else if len(datestr) >= len("13980450781991") {
			if microSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(0, microSecs*1000), nil
			} else {
				//u.Error(err)
			}
		} else {
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				return time.Unix(secs, 0), nil
			} else {
				//u.Error(err)
			}
		}
	case ST_DIGITDASH: // starts digit then dash 02-
		// 2006-01-02
		if len(datestr) == len("2014-04-26") {
			if t, err := time.Parse("2006-01-02", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		}
	case ST_DIGITDASHWS: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2014-04-26 17:24:37.3186369
		// 2012-08-03 18:31:59.257000000
		// 2016-03-14 00:00:00.000
		// 2013-04-01 22:43:22
		if len(datestr) == len("2012-08-03 18:31:59.257000000") {
			if t, err := time.Parse("2006-01-02 15:04:05.000000000", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		} else if len(datestr) == len("2014-04-26 05:24:37.3186369") {
			if t, err := time.Parse("2006-01-02 15:04:05.0000000", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		} else if len(datestr) == len("2014-04-26 05:24:37.000") {
			if t, err := time.Parse("2006-01-02 15:04:05.000", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		} else if len(datestr) == len("2013-04-01 22:43:22") {
			if t, err := time.Parse("2006-01-02 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		}
	case ST_DIGITSLASH: // starts digit then slash 02/ (but nothing else)
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006

		if len(datestr) == len("01/02/2006") {
			if t, err := time.Parse("01/02/2006", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		} else {
			if t, err := time.Parse("1/2/2006", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		}

	case ST_DIGITSLASHWSCOLON: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05
		if len(datestr) == len("01/02/2006 15:04") {
			if t, err := time.Parse("01/02/2006 15:04", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		} else if len(datestr) == len("01/2/2006 15:04") {
			if t, err := time.Parse("01/2/2006 15:04", datestr); err == nil {
				return t, nil
			} else {
				if t, err := time.Parse("1/02/2006 15:04", datestr); err == nil {
					return t, nil
				} else {
					//u.Error(err)
				}
			}
		} else {
			if t, err := time.Parse("1/2/2006 15:04", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		}
	case ST_DIGITSLASHWSCOLONCOLON: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59
		// 03/1/2012 10:11:59
		// 3/01/2012 10:11:59
		if len(datestr) == len("01/02/2006 15:04:05") {
			if t, err := time.Parse("01/02/2006 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
			}
		} else if len(datestr) == len("01/2/2006 15:04:05") {
			if t, err := time.Parse("01/2/2006 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				if t, err := time.Parse("1/02/2006 15:04:05", datestr); err == nil {
					return t, nil
				} else {
					//u.Error(err)
					return time.Time{}, err
				}
			}
		} else {
			if t, err := time.Parse("1/2/2006 15:04:05", datestr); err == nil {
				return t, nil
			} else {
				//u.Error(err)
				return time.Time{}, err
			}
		}
	case ST_ALPHACOMMA: // Starts alpha then comma but no DASH
		// Mon, 02 Jan 2006 15:04:05 MST
		if t, err := time.Parse("Jan 2, 2006 3:04:05 PM", datestr); err == nil {
			return t, nil
		} else {
			//u.Errorf("'%s'  err=%v", datestr, err)
			return time.Time{}, err
		}
	case ST_ALPHACOMMADASH: // Starts alpha then comma and one dash
		// Mon, 02 Jan 2006 15:04:05 -0700

		//RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		// TODO:  this doesn't work???
		if t, err := time.Parse(time.RFC1123Z, datestr); err == nil {
			return t, nil
		} else {
			//u.Errorf("'%s'  err=%v", datestr, err)
			return time.Time{}, err
		}

	case ST_ALPHACOMMADASHDASH: // Starts alpha then comma and two dash'es
		// Monday, 02-Jan-06 15:04:05 MST
		if t, err := time.Parse("Monday, 02-Jan-06 15:04:05 MST", datestr); err == nil {
			return t, nil
		} else {
			//u.Errorf("'%s'  err=%v", datestr, err)
			return time.Time{}, err
		}
	default:
		//u.Infof("no case for: %d : %s", state, datestr)
	}

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}
