package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/tkuchiki/parsetime"
)

func main() {
	if len(os.Args) <= 1 {
		q := Time2Qreki(time.Now())
		fmt.Printf("今日は旧暦の%sです。%s\n", q, q.Rokuyou().Explanation())
	} else {
		t, err := parsetime.Parse(os.Args[1])
		if err != nil {
			fmt.Println("日付を入力して下さい")
			return
		}
		q := Time2Qreki(t)
		fmt.Printf("%sは旧暦の%sです。%s\n", t.Format("2006-01-02"), q, q.Rokuyou().Explanation())
	}
}

type Rokuyou int

func (r Rokuyou) String() string {
	switch r {
	case 0:
		return "大安"
	case 1:
		return "赤口"
	case 2:
		return "先勝"
	case 3:
		return "友引"
	case 4:
		return "先負"
	case 5:
		return "仏滅"
	}
	return ""
}

func (r Rokuyou) Explanation() string {
	switch r {
	case 0:
		return "思い切ってdeployしちゃいましょう。"
	case 1:
		return "実は仏滅よりもやばいです。deployしたらあかん..."
	case 2:
		return "deployは午前中に済ませましょう。"
	case 3:
		return "昼のdeployはさけましょう。するなら朝晩が吉です。"
	case 4:
		return "deployは午後からが吉でしょう。"
	case 5:
		return "仏滅deployとか事故のもとですよ。"
	}
	return ""
}

type Qreki struct {
	Month     int
	Day       int
	LeapMonth bool
}

func (q Qreki) Rokuyou() Rokuyou {
	return Rokuyou((q.Month + q.Day) % 6)
}

func (q Qreki) String() string {
	leapMonth := ""
	if q.LeapMonth {
		leapMonth = "閏"
	}
	return fmt.Sprintf("%s%d月%d日(%s)", leapMonth, q.Month, q.Day, q.Rokuyou())
}

// 旧暦の各月の朔日を表す
type FirstDay struct {
	d            time.Time
	sunLongitude float64
}

func Time2Qreki(now time.Time) Qreki {
	// 直近の旧暦11月1日(直近の冬至の直近の朔)を求める
	t := PreviousSaku(PreviousTouji(now))

	// 旧暦11月1日から各月の朔日を求める
	firstDays := [14]FirstDay{}
	for i := 0; i < 14; i++ {
		firstDays[i] = FirstDay{t, SunLongitude(Time2JulianYear(t))}
		t = NextSaku(t)
	}

	if firstDays[13].sunLongitude < 270 {
		// 閏月が存在する
		m := 10
		leapFlag := false
		for i := 0; i < 13; i++ {
			leapMonth := false
			if !leapFlag && int(firstDays[i].sunLongitude/30) == int(firstDays[i+1].sunLongitude/30) {
				// 中気を含まない最初の月は閏月
				leapFlag = true
				leapMonth = true
			} else {
				m++
				if m > 12 {
					m = 1
				}
			}

			if firstDays[i+1].d.After(now) {
				return Qreki{
					Month:     m,
					Day:       int(now.Sub(firstDays[i].d)/(24*time.Hour)) + 1,
					LeapMonth: leapMonth,
				}
			}
		}
	} else {
		// 閏月は存在しない
		for i := 0; i < 13; i++ {
			if firstDays[i+1].d.After(now) {
				return Qreki{
					Month:     (i+10)%12 + 1,
					Day:       int(now.Sub(firstDays[i].d)/(24*time.Hour)) + 1,
					LeapMonth: false,
				}
			}
		}
	}

	panic(errors.New("something wrong"))
}

func PreviousSaku(t time.Time) time.Time {
	return Previous(t, func(jy JulianYear) float64 {
		return normalizeDegree(MoonLongitude(jy) - SunLongitude(jy))
	}, 4812.67881-360.00769)
}

func NextSaku(t time.Time) time.Time {
	return Next(t, func(jy JulianYear) float64 {
		return normalizeDegree(MoonLongitude(jy) - SunLongitude(jy))
	}, 4812.67881-360.00769)
}

func PreviousTouji(t time.Time) time.Time {
	return Previous(t, func(jy JulianYear) float64 {
		return normalizeDegree(SunLongitude(jy) - 270)
	}, 360.00769)
}

// 前回関数fが0になった日を求める
func Previous(t time.Time, f func(jy JulianYear) float64, a float64) time.Time {
	// 傾きaを使って大雑把に予測
	y, m, d := t.Date()
	t = time.Date(y, m, d, 0, 0, 0, 0, JST).Add(24 * time.Hour)
	jyt := Time2JulianYear(t)
	ft := f(jyt)
	t = (jyt - JulianYear(ft/a)).Time()

	// 予測があたっているか確かめる
	y, m, d = t.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, JST)
	end := start.Add(24 * time.Hour)

	jyStart := Time2JulianYear(start)
	jyEnd := Time2JulianYear(end)
	fStart := f(jyStart)
	fEnd := f(jyEnd)

	if fStart >= fEnd {
		return start
	} else if fEnd < ft {
		return previous(start, f)
	} else {
		return next(start, f)
	}
}

// 次回関数fが0になる日を求める
func Next(t time.Time, f func(jy JulianYear) float64, a float64) time.Time {
	// 傾きaを使って大雑把に予測
	y, m, d := t.Date()
	t = time.Date(y, m, d, 0, 0, 0, 0, JST).Add(24 * time.Hour)
	jyt := Time2JulianYear(t)
	ft := f(jyt)
	t = (jyt + JulianYear((360-ft)/a)).Time()

	// 予測があたっているか確かめる
	y, m, d = t.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, JST)
	end := start.Add(24 * time.Hour)

	jyStart := Time2JulianYear(start)
	jyEnd := Time2JulianYear(end)
	fStart := f(jyStart)
	fEnd := f(jyEnd)

	if fStart >= fEnd {
		return start
	} else if fEnd < ft {
		return previous(start, f)
	} else {
		return next(start, f)
	}
}

// 前回関数fが0になった日を求める(線形検索)
func previous(t time.Time, f func(jy JulianYear) float64) time.Time {
	y, m, d := t.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, JST)
	end := start.Add(24 * time.Hour)

	jyStart := Time2JulianYear(start)
	jyEnd := Time2JulianYear(end)
	fStart := f(jyStart)
	fEnd := f(jyEnd)

	for fStart < fEnd {
		start, end = start.Add(-24*time.Hour), start
		jyStart, jyEnd = Time2JulianYear(start), jyStart
		fStart, fEnd = f(jyStart), fStart
	}

	return start
}

// 次回関数fが0になる日を求める
func next(t time.Time, f func(jy JulianYear) float64) time.Time {
	y, m, d := t.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, JST).Add(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	jyStart := Time2JulianYear(start)
	jyEnd := Time2JulianYear(end)
	fStart := f(jyStart)
	fEnd := f(jyEnd)

	for fStart < fEnd {
		start, end = end, end.Add(24*time.Hour)
		jyStart, jyEnd = jyEnd, Time2JulianYear(end)
		fStart, fEnd = fEnd, f(jyEnd)
	}

	return start
}

// cited from 長沢 工(1999) "日の出・日の入りの計算 天体の出没時刻の求め方" 株式会社地人書館
var moonLongitudeTable = [...][3]float64{
	{1.2740, 100.738, 4133.3536},
	{0.6583, 235.700, 8905.3422},
	{0.2136, 269.926, 9543.9773},
	{0.1856, 177.525, 359.9905},
	{0.1143, 6.546, 9664.0404},
	{0.0588, 214.22, 638.635},
	{0.0572, 103.21, 3773.363},
	{0.0533, 10.66, 13677.331},
	{0.0459, 238.18, 8545.352},
	{0.0410, 137.43, 4411.998},
	{0.0348, 117.84, 4452.671},
	{0.0305, 312.49, 5131.979},
	{0.0153, 130.84, 758.698},
	{0.0125, 141.51, 14436.029},
	{0.0110, 231.59, 4892.052},
	{0.0107, 336.44, 13038.696},
	{0.0100, 44.89, 14315.966},
	{0.0085, 201.5, 8266.71},
	{0.0079, 278.2, 4493.34},
	{0.0068, 53.2, 9265.33},
	{0.0052, 197.2, 319.32},
	{0.0050, 295.4, 4812.66},
	{0.0048, 235.0, 19.34},
	{0.0040, 13.2, 13317.34},
	{0.0040, 145.6, 18449.32},
	{0.0040, 119.5, 1.33},
	{0.0039, 111.3, 17810.68},
	{0.0037, 349.1, 5410.62},
	{0.0027, 272.5, 9183.99},
	{0.0026, 107.2, 13797.39},
	{0.0024, 211.9, 988.63},
	{0.0024, 252.8, 9224.66},
	{0.0022, 240.6, 8185.36},
	{0.0021, 87.5, 9903.97},
	{0.0021, 175.1, 719.98},
	{0.0021, 105.6, 3413.37},
	{0.0020, 55.0, 19.34},
	{0.0018, 4.1, 4013.29},
	{0.0016, 242.2, 18569.38},
	{0.0012, 339.0, 12678.71},
	{0.0011, 276.5, 19208.02},
	{0.0009, 218, 8586.0},
	{0.0008, 188, 14037.3},
	{0.0008, 204, 7906.7},
	{0.0007, 140, 4052.0},
	{0.0007, 275, 4853.3},
	{0.0007, 216, 278.6},
	{0.0006, 128, 1118.7},
	{0.0005, 247, 22582.7},
	{0.0005, 181, 19088.0},
	{0.0005, 114, 17450.7},
	{0.0005, 332, 5091.3},
	{0.0004, 313, 398.7},
	{0.0004, 278, 120.1},
	{0.0004, 71, 9584.7},
	{0.0004, 20, 720.0},
	{0.0003, 83, 3814.0},
	{0.0003, 66, 3494.7},
	{0.0003, 147, 18089.3},
	{0.0003, 311, 5492.0},
	{0.0003, 161, 40.7},
	{0.0003, 280, 23221.3},
}

func MoonLongitude(julianYear JulianYear) float64 {
	t := float64(julianYear)
	a := 0.0040*sin(119.5+1.33*t) +
		0.0020*sin(55.0+19.34*t) +
		0.0006*sin(71+0.2*t) +
		0.0006*sin(54+19.3*t)
	l := normalizeDegree(218.3161 + 4812.67881*t + 6.2887*sin(134.961+4771.9886*t+a))
	for _, b := range moonLongitudeTable {
		l = normalizeDegree(l + b[0]*sin(b[1]+b[2]*t))
	}
	return l
}

var sunLongitudeTable = [...][3]float64{
	{0.0200, 355.05, 719.981},
	{0.0048, 234.95, 19.341},
	{0.0020, 247.1, 329.64},
	{0.0018, 297.8, 4452.67},
	{0.0018, 251.3, 0.20},
	{0.0015, 343.2, 450.37},
	{0.0013, 81.4, 225.18},
	{0.0008, 132.5, 659.29},
	{0.0007, 153.3, 90.38},
	{0.0007, 206.8, 30.35},
	{0.0006, 29.8, 337.18},
	{0.0005, 207.4, 1.50},
	{0.0005, 291.2, 22.81},
	{0.0004, 234.9, 315.56},
	{0.0004, 157.3, 299.30},
	{0.0004, 21.1, 720.02},
	{0.0003, 352.5, 1079.97},
	{0.0003, 329.7, 44.43},
}

func SunLongitude(julianYear JulianYear) float64 {
	t := float64(julianYear)
	l := normalizeDegree(280.4603 + 360.00769*t + (1.9146-0.00005*t)*sin(357.538+359.991*t))
	for _, b := range sunLongitudeTable {
		l = normalizeDegree(l + b[0]*sin(b[1]+b[2]*t))
	}
	return l
}

var JST, _ = time.LoadLocation("Asia/Tokyo")

// JulianYear is a number of julian years from J2000.0(2000/01/01 12:00 Terrestrial Time)
type JulianYear float64

var j2000 = time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)

func Time2JulianYear(t time.Time) JulianYear {
	d := t.Sub(j2000)

	// convert UTC(Coordinated Universal Time) into TAI(International Atomic Time)
	d += 36 * time.Second // TAI - UTC = 36seconds (at 2015/08)

	// convert TAI into TT(Terrestrial Time)
	d += 32184 * time.Millisecond
	return JulianYear(float64(d) / float64((365*24+6)*time.Hour))
}

func (jy JulianYear) Time() time.Time {
	// convert TT into TAI
	d := time.Duration(float64(jy) * float64((365*24+6)*time.Hour))
	d -= 32184 * time.Millisecond
	t := j2000.Add(d)

	// convert TAI into UTC
	t = t.Add(-36 * time.Second)

	return t
}

func sin(x float64) float64 {
	return math.Sin(x / 180 * math.Pi)
}

func normalizeDegree(x float64) float64 {
	x = math.Mod(x, 360)
	if x < 0 {
		x += 360
	}
	return x
}
