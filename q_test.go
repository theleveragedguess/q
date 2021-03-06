package q

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/itsubaki/q/pkg/math/number"

	"github.com/itsubaki/q/pkg/math/matrix"
	"github.com/itsubaki/q/pkg/quantum/gate"
	"github.com/itsubaki/q/pkg/quantum/qubit"
)

func TestQSimFactoring15(t *testing.T) {
	N := 15
	a := 7

	// co-prime
	if number.GCD(N, a) != 1 {
		t.Errorf("%v %v\n", N, a)
	}

	var count int
	for {
		count++

		qsim := New()

		q0 := qsim.Zero()
		q1 := qsim.Zero()
		q2 := qsim.Zero()

		q3 := qsim.Zero()
		q4 := qsim.Zero()
		q5 := qsim.Zero()
		q6 := qsim.One()

		// superposition
		qsim.H(q0, q1, q2)

		// Controlled-U
		qsim.CNOT(q2, q4)
		qsim.CNOT(q2, q5)

		// Controlled-U^2
		qsim.ControlledNot([]Qubit{q1, q4}, q6)
		qsim.ControlledNot([]Qubit{q1, q6}, q4)
		qsim.ControlledNot([]Qubit{q1, q4}, q6)

		qsim.ControlledNot([]Qubit{q1, q3}, q5)
		qsim.ControlledNot([]Qubit{q1, q5}, q3)
		qsim.ControlledNot([]Qubit{q1, q3}, q5)

		// QFT
		qsim.H(q0)
		qsim.CR(q1, q0, 2)
		qsim.CR(q2, q0, 3)
		qsim.H(q1)
		qsim.CR(q2, q1, 2)
		qsim.H(q2)
		qsim.Swap(q0, q2)

		// measure q0, q1, q2
		qsim.Measure(q0)
		qsim.Measure(q1)
		qsim.Measure(q2)

		// measure q3, q4, q5, q6
		qsim.Measure(q3)
		qsim.Measure(q4)
		qsim.Measure(q5)
		qsim.Measure(q6)

		// get integer
		bin := qsim.Binary()
		dec, err := strconv.ParseInt(bin[3:], 2, 64)
		if err != nil {
			t.Errorf("parse int: %v", err)
		}

		// probability is
		// 010,0001(1)  0.25
		// 010,0100(4)  0.25
		// 010,0111(7)  0.25
		// 010,1101(13) 0.25

		// continued fraction
		_, _, r := number.Fraction(float64(dec)/16.0, 1e-3)
		if r > N || number.IsOdd(r) {
			continue
		}

		// gcd(a^(r/2)-1, N)
		// gcd(a^(r/2)+1, N)
		p0 := number.GCD(number.Pow(a, r/2)-1, N)
		p1 := number.GCD(number.Pow(a, r/2)+1, N)

		if p0*p1 != N {
			continue
		}

		fmt.Printf("N=%d, p=%v, q=%v. count=%d\n", N, p0, p1, count)
		break
	}
}

func TestQSimGrover3qubit(t *testing.T) {
	print := func(q *Q) {
		p := q.Amplitude()
		n := q.NumberOfBit()
		f := "%0" + strconv.Itoa(n) + "s %1.3f, "

		var out string
		for i := range p {
			bit := strconv.FormatInt(int64(i), 2)
			out = out + fmt.Sprintf(f, bit, p[i])
		}
		out = out[:len(out)-2] // remove last ,
		fmt.Println(out)
	}

	qsim := New()

	q0 := qsim.Zero()
	q1 := qsim.Zero()
	q2 := qsim.Zero()
	q3 := qsim.One()

	qsim.H(q0, q1, q2, q3)
	print(qsim)

	// oracle
	qsim.X(q0).ControlledNot([]Qubit{q0, q1, q2}, q3).X(q0)
	print(qsim)

	// amp
	qsim.H(q0, q1, q2, q3)
	qsim.X(q0, q1, q2)
	qsim.ControlledZ([]Qubit{q0, q1}, q2)
	qsim.H(q0, q1, q2)
	print(qsim)

	// q3 is always |1>
	m3 := qsim.Measure(q3)
	if !m3.IsOne() {
		t.Error(m3)
	}

	p := qsim.Probability()
	for i, pp := range p {
		// |011>|1>
		if i == 7 && math.Abs(pp-0.78125) > 1e-13 {
			t.Error(qsim.Probability())
		}

		if i%2 == 0 && math.Abs(pp) > 1e-13 {
			t.Error(qsim.Probability())
		}

		if i%2 == 1 && i != 7 && math.Abs(pp-0.03125) > 1e-13 {
			t.Error(qsim.Probability())
		}

	}

	for i, pp := range p {
		if pp == 0 {
			continue
		}

		fmt.Printf("%04s %v\n", strconv.FormatInt(int64(i), 2), pp)
	}
}

func TestApply(t *testing.T) {
	qsim := New()

	i0 := qsim.Zero().Index()
	i1 := qsim.Zero().Index()
	n := qsim.NumberOfBit()

	g0 := gate.H().TensorProduct(gate.I())
	g1 := gate.CNOT(n, i0, i1)
	qc := g0.Apply(g1)

	qsim.Apply(qc)

	if math.Abs(qsim.Probability()[0]-0.5) > 1e13 {
		t.Error(qsim.Probability())
	}

	if math.Abs(qsim.Probability()[1]) > 1e13 {
		t.Error(qsim.Probability())
	}

	if math.Abs(qsim.Probability()[2]) > 1e13 {
		t.Error(qsim.Probability())
	}

	if math.Abs(qsim.Probability()[3]-0.5) > 1e13 {
		t.Error(qsim.Probability())
	}
}

func TestPOVM(t *testing.T) {
	E1 := gate.New(
		[]complex128{0, 0},
		[]complex128{0, 1},
	).Mul(complex(math.Sqrt(2)/(1.0+math.Sqrt(2)), 0))

	E2 := gate.New(
		[]complex128{1, -1},
		[]complex128{-1, 1},
	).Mul(complex(0.5, 0)).
		Mul(complex(math.Sqrt(2)/(1.0+math.Sqrt(2)), 0))

	E3 := gate.I().Sub(E1).Sub(E2)

	if !E1.Add(E2).Add(E3).Equals(gate.I()) {
		t.Fail()
	}

	q0 := qubit.Zero()
	if q0.Apply(E1).InnerProduct(q0) != complex(0, 0) {
		t.Fail()
	}

	q1 := qubit.Zero().Apply(gate.H())
	if q1.Apply(E2).InnerProduct(q1) != complex(0, 0) {
		t.Fail()
	}
}

func TestQSimQFT(t *testing.T) {
	qsim := New()

	qsim.Zero()
	qsim.Zero()
	qsim.Zero()

	qsim.QFT()

	p := qsim.Probability()
	for _, pp := range p {
		if math.Abs(pp-0.125) > 1e-13 {
			t.Error(p)
		}
	}
}

func TestQSimInverseQFT(t *testing.T) {
	qsim := New()

	qsim.Zero()
	qsim.Zero()
	qsim.Zero()

	qsim.InverseQFT()

	p := qsim.Probability()
	for _, pp := range p {
		if math.Abs(pp-0.125) > 1e-13 {
			t.Error(p)
		}
	}
}

func TestQSimQFT3qubit(t *testing.T) {
	qsim := New()

	q0 := qsim.Zero()
	q1 := qsim.Zero()
	q2 := qsim.Zero()

	qsim.H(q0)
	qsim.CR(q1, q0, 2)
	qsim.CR(q2, q0, 3)

	qsim.H(q1)
	qsim.CR(q2, q1, 2)

	qsim.H(q2)

	qsim.Swap(q0, q2)

	p := qsim.Probability()
	for _, pp := range p {
		if math.Abs(pp-0.125) > 1e-13 {
			t.Error(p)
		}
	}
}

func TestQSimCNOT(t *testing.T) {
	qsim := New()

	q0 := qsim.Zero()
	q1 := qsim.Zero()

	p := qsim.ControlledNot([]Qubit{q0}, q1).Probability()
	e := qubit.Zero(2).Apply(gate.CNOT(2, 0, 1)).Probability()

	for i := range p {
		if p[i] != e[i] {
			t.Errorf("%v: %v\n", p, e)
		}
	}
}

func TestQSimEstimate(t *testing.T) {
	qsim := New()

	q0 := qsim.Zero()
	q1 := qsim.Zero()

	qsim.H(q0, q1)

	for _, p := range qsim.Probability() {
		if math.Abs(p-0.25) > 1e-13 {
			t.Error(qsim.Probability())
		}
	}

	ex := qubit.Zero().Apply(gate.H())
	e0 := qsim.Estimate(q0)
	e1 := qsim.Estimate(q1)

	f0 := ex.Fidelity(e0)
	f1 := ex.Fidelity(e1)

	if math.Abs(f0-1) > 1e-2 {
		t.Errorf("%v: %v\n", f0, e0)
	}

	if math.Abs(f1-1) > 1e-2 {
		t.Errorf("%v: %v\n", f1, e1)
	}
}

func TestQSimBellState(t *testing.T) {
	qsim := New()

	q0 := qsim.Zero()
	q1 := qsim.Zero()

	qsim.H(q0).CNOT(q0, q1)

	p := qsim.Probability()
	if math.Abs(qubit.Sum(p)-1) > 1e-13 {
		t.Error(p)
	}

	if math.Abs(p[0]-0.5) > 1e-13 {
		t.Error(p)
	}

	if math.Abs(p[3]-0.5) > 1e-13 {
		t.Error(p)
	}

	if qsim.Measure(q0).IsZero() && qsim.Measure(q1).IsOne() {
		t.Error(qsim.Probability())
	}

	if qsim.Measure(q0).IsOne() && qsim.Measure(q1).IsZero() {
		t.Error(qsim.Probability())
	}
}

func TestQsimQuantumTeleportation2(t *testing.T) {
	qsim := New()

	phi := qsim.New(1, 2)
	q0 := qsim.Zero()
	q1 := qsim.Zero()

	qsim.H(q0).CNOT(q0, q1) // bell state
	qsim.CNOT(phi, q0).H(phi)

	qsim.CNOT(q0, q1)
	qsim.CZ(phi, q1)

	mz := qsim.Measure(phi)
	mx := qsim.Measure(q0)

	p := qsim.Probability()
	if math.Abs(qubit.Sum(p)-1) > 1e-13 {
		t.Error(p)
	}

	var test = []struct {
		zero int
		one  int
		zval float64
		oval float64
		eps  float64
		mz   *qubit.Qubit
		mx   *qubit.Qubit
	}{
		{0, 1, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.Zero()},
		{2, 3, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.One()},
		{4, 5, 0.2, 0.8, 1e-13, qubit.One(), qubit.Zero()},
		{6, 7, 0.2, 0.8, 1e-13, qubit.One(), qubit.One()},
	}

	for _, tt := range test {
		if p[tt.zero] == 0 {
			continue
		}

		if math.Abs(p[tt.zero]-tt.zval) > tt.eps {
			t.Error(p)
		}
		if math.Abs(p[tt.one]-tt.oval) > tt.eps {
			t.Error(p)
		}

		if !mz.Equals(tt.mz) {
			t.Error(p)
		}

		if !mx.Equals(tt.mx) {
			t.Error(p)
		}

	}
}

func TestQSimQuantumTeleportation(t *testing.T) {
	qsim := New()

	phi := qsim.New(1, 2)
	q0 := qsim.Zero()
	q1 := qsim.Zero()

	qsim.H(q0).CNOT(q0, q1) // bell state
	qsim.CNOT(phi, q0).H(phi)

	mz := qsim.Measure(phi)
	mx := qsim.Measure(q0)

	qsim.ConditionZ(mz.IsOne(), q1)
	qsim.ConditionX(mx.IsOne(), q1)

	p := qsim.Probability()
	if math.Abs(qubit.Sum(p)-1) > 1e-13 {
		t.Error(p)
	}

	var test = []struct {
		zero int
		one  int
		zval float64
		oval float64
		eps  float64
		mz   *qubit.Qubit
		mx   *qubit.Qubit
	}{
		{0, 1, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.Zero()},
		{2, 3, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.One()},
		{4, 5, 0.2, 0.8, 1e-13, qubit.One(), qubit.Zero()},
		{6, 7, 0.2, 0.8, 1e-13, qubit.One(), qubit.One()},
	}

	for _, tt := range test {
		if p[tt.zero] == 0 {
			continue
		}

		if math.Abs(p[tt.zero]-tt.zval) > tt.eps {
			t.Error(p)
		}
		if math.Abs(p[tt.one]-tt.oval) > tt.eps {
			t.Error(p)
		}

		if !mz.Equals(tt.mz) {
			t.Error(p)
		}

		if !mx.Equals(tt.mx) {
			t.Error(p)
		}

	}
}

func TestQsimErrorCorrectionZero(t *testing.T) {
	qsim := New()

	q0 := qsim.Zero()

	// encoding
	q1 := qsim.Zero()
	q2 := qsim.Zero()
	qsim.CNOT(q0, q1).CNOT(q0, q2)

	// error: first qubit is flipped
	qsim.X(q0)

	// add ancilla qubit
	q3 := qsim.Zero()
	q4 := qsim.Zero()

	// error correction
	qsim.CNOT(q0, q3).CNOT(q1, q3)
	qsim.CNOT(q1, q4).CNOT(q2, q4)

	m3 := qsim.Measure(q3)
	m4 := qsim.Measure(q4)

	qsim.ConditionX(m3.IsOne() && m4.IsZero(), q0)
	qsim.ConditionX(m3.IsOne() && m4.IsOne(), q1)
	qsim.ConditionX(m3.IsZero() && m4.IsOne(), q2)

	// |q0q1q2> = |000>
	if !qsim.Estimate(q0).IsZero() {
		t.Error(qsim.Estimate(q0))
	}

	if !qsim.Estimate(q1).IsZero() {
		t.Error(qsim.Estimate(q1))
	}

	if !qsim.Estimate(q2).IsZero() {
		t.Error(qsim.Estimate(q2))
	}

	// |000>|10>
	if qsim.Probability()[2] != 1 {
		t.Error(qsim.Probability())
	}
}

func TestQsimErrorCorrection(t *testing.T) {
	qsim := New()

	q0 := qsim.New(1, 9)

	// encoding
	q1 := qsim.Zero()
	q2 := qsim.Zero()
	qsim.CNOT(q0, q1).CNOT(q0, q2)

	// error: first qubit is flipped
	qsim.X(q1)

	// add ancilla qubit
	q3 := qsim.Zero()
	q4 := qsim.Zero()

	// error correction
	qsim.CNOT(q0, q3).CNOT(q1, q3)
	qsim.CNOT(q1, q4).CNOT(q2, q4)

	m3 := qsim.Measure(q3)
	m4 := qsim.Measure(q4)

	qsim.ConditionX(m3.IsOne() && m4.IsZero(), q0)
	qsim.ConditionX(m3.IsOne() && m4.IsOne(), q1)
	qsim.ConditionX(m3.IsZero() && m4.IsOne(), q2)

	ex := qubit.New(1, 9)
	f0 := ex.Fidelity(qsim.Estimate(q0))
	f1 := ex.Fidelity(qsim.Estimate(q1))
	f2 := ex.Fidelity(qsim.Estimate(q2))

	if math.Abs(f0-1) > 1e-3 {
		t.Errorf("%v\n", f0)
	}

	if math.Abs(f1-1) > 1e-3 {
		t.Errorf("%v\n", f1)
	}

	if math.Abs(f2-1) > 1e-3 {
		t.Errorf("%v\n", f2)
	}
}

func TestGrover3qubit(t *testing.T) {
	x := matrix.TensorProduct(gate.X(), gate.I(3))
	oracle := x.Apply(gate.ControlledNot(4, []int{0, 1, 2}, 3)).Apply(x)

	h4 := matrix.TensorProduct(gate.H(3), gate.H())
	x3 := matrix.TensorProduct(gate.X(3), gate.I())
	cz := matrix.TensorProduct(gate.ControlledZ(3, []int{0, 1}, 2), gate.I())
	h3 := matrix.TensorProduct(gate.H(3), gate.I())
	amp := h4.Apply(x3).Apply(cz).Apply(x3).Apply(h3)

	q := qubit.TensorProduct(qubit.Zero(3), qubit.One())
	q.Apply(gate.H(4)).Apply(oracle).Apply(amp)

	// q3 is always |1>
	q3 := q.Measure(3)
	if !q3.IsOne() {
		t.Error(q3)
	}

	p := q.Probability()
	for i, pp := range p {
		// |011>|1>
		if i == 7 {
			if math.Abs(pp-0.78125) > 1e-13 {
				t.Error(q.Probability())
			}
			continue
		}

		if i%2 == 0 {
			if math.Abs(pp) > 1e-13 {
				t.Error(q.Probability())
			}
			continue
		}

		if math.Abs(pp-0.03125) > 1e-13 {
			t.Error(q.Probability())
		}
	}
}

func TestGrover2qubit(t *testing.T) {
	oracle := gate.CZ(2, 0, 1)

	h2 := gate.H(2)
	x2 := gate.X(2)
	amp := h2.Apply(x2).Apply(gate.CZ(2, 0, 1)).Apply(x2).Apply(h2)

	qc := h2.Apply(oracle).Apply(amp)
	q := qubit.Zero(2).Apply(qc)

	q.Measure()
	if math.Abs(q.Probability()[3]-1) > 1e-13 {
		t.Error(q.Probability())
	}
}

func TestQuantumTeleportation(t *testing.T) {
	g0 := matrix.TensorProduct(gate.H(), gate.I())
	g1 := gate.CNOT(2, 0, 1)
	bell := qubit.Zero(2).Apply(g0).Apply(g1)

	phi := qubit.New(1, 2)
	phi.TensorProduct(bell)

	g2 := gate.CNOT(3, 0, 1)
	g3 := matrix.TensorProduct(gate.H(), gate.I(2))
	phi.Apply(g2).Apply(g3)

	mz := phi.Measure(0)
	mx := phi.Measure(1)

	if mz.IsOne() {
		z := matrix.TensorProduct(gate.I(2), gate.Z())
		phi.Apply(z)
	}

	if mx.IsOne() {
		x := matrix.TensorProduct(gate.I(2), gate.X())
		phi.Apply(x)
	}

	var test = []struct {
		zero int
		one  int
		zval float64
		oval float64
		eps  float64
		mz   *qubit.Qubit
		mx   *qubit.Qubit
	}{
		{0, 1, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.Zero()},
		{2, 3, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.One()},
		{4, 5, 0.2, 0.8, 1e-13, qubit.One(), qubit.Zero()},
		{6, 7, 0.2, 0.8, 1e-13, qubit.One(), qubit.One()},
	}

	p := phi.Probability()
	if math.Abs(qubit.Sum(p)-1) > 1e-13 {
		t.Error(p)
	}

	for _, tt := range test {
		if p[tt.zero] == 0 {
			continue
		}

		if math.Abs(p[tt.zero]-tt.zval) > tt.eps {
			t.Error(p)
		}
		if math.Abs(p[tt.one]-tt.oval) > tt.eps {
			t.Error(p)
		}

		if !mz.Equals(tt.mz) {
			t.Error(p)
		}

		if !mx.Equals(tt.mx) {
			t.Error(p)
		}
	}
}

func TestQuantumTeleportation2(t *testing.T) {
	g0 := matrix.TensorProduct(gate.H(), gate.I())
	g1 := gate.CNOT(2, 0, 1)
	bell := qubit.Zero(2).Apply(g0).Apply(g1)

	phi := qubit.New(1, 2)
	phi.TensorProduct(bell)

	g2 := gate.CNOT(3, 0, 1)
	g3 := matrix.TensorProduct(gate.H(), gate.I(2))
	g4 := gate.CNOT(3, 1, 2)
	g5 := gate.CZ(3, 0, 2)

	phi.Apply(g2).Apply(g3).Apply(g4).Apply(g5)

	mz := phi.Measure(0)
	mx := phi.Measure(1)

	var test = []struct {
		zero int
		one  int
		zval float64
		oval float64
		eps  float64
		mz   *qubit.Qubit
		mx   *qubit.Qubit
	}{
		{0, 1, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.Zero()},
		{2, 3, 0.2, 0.8, 1e-13, qubit.Zero(), qubit.One()},
		{4, 5, 0.2, 0.8, 1e-13, qubit.One(), qubit.Zero()},
		{6, 7, 0.2, 0.8, 1e-13, qubit.One(), qubit.One()},
	}

	p := phi.Probability()
	if math.Abs(qubit.Sum(p)-1) > 1e-13 {
		t.Error(p)
	}

	for _, tt := range test {
		if p[tt.zero] == 0 {
			continue
		}

		if math.Abs(p[tt.zero]-tt.zval) > tt.eps {
			t.Error(p)
		}

		if math.Abs(p[tt.one]-tt.oval) > tt.eps {
			t.Error(p)
		}

		if !mz.Equals(tt.mz) {
			t.Error(p)
		}

		if !mx.Equals(tt.mx) {
			t.Error(p)
		}
	}
}

func TestErrorCorrectionZero(t *testing.T) {
	phi := qubit.Zero()

	// encoding
	phi.TensorProduct(qubit.Zero(2))
	phi.Apply(gate.CNOT(3, 0, 1))
	phi.Apply(gate.CNOT(3, 0, 2))

	// error: first qubit is flipped
	phi.Apply(matrix.TensorProduct(gate.X(), gate.I(2)))

	// add ancilla qubit
	phi.TensorProduct(qubit.Zero(2))

	// z1z2
	phi.Apply(gate.CNOT(5, 0, 3)).Apply(gate.CNOT(5, 1, 3))

	// z2z3
	phi.Apply(gate.CNOT(5, 1, 4)).Apply(gate.CNOT(5, 2, 4))

	// measure
	m3 := phi.Measure(3)
	m4 := phi.Measure(4)

	// recover
	if m3.IsOne() && m4.IsZero() {
		phi.Apply(matrix.TensorProduct(gate.X(), gate.I(4)))
	}

	if m3.IsOne() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(), gate.X(), gate.I(3)))
	}

	if m3.IsZero() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(2), gate.X(), gate.I(2)))
	}

	// answer is |000>|10>
	if phi.Probability()[2] != 1 {
		t.Error(phi.Probability())
	}
}

func TestErrorCorrectionOne(t *testing.T) {
	phi := qubit.One()

	// encoding
	phi.TensorProduct(qubit.Zero(2))
	phi.Apply(gate.CNOT(3, 0, 1))
	phi.Apply(gate.CNOT(3, 0, 2))

	// error: first qubit is flipped
	phi.Apply(matrix.TensorProduct(gate.X(), gate.I(2)))

	// add ancilla qubit
	phi.TensorProduct(qubit.Zero(2))

	// z1z2
	phi.Apply(gate.CNOT(5, 0, 3)).Apply(gate.CNOT(5, 1, 3))

	// z2z3
	phi.Apply(gate.CNOT(5, 1, 4)).Apply(gate.CNOT(5, 2, 4))

	// measure
	m3 := phi.Measure(3)
	m4 := phi.Measure(4)

	// recover
	if m3.IsOne() && m4.IsZero() {
		phi.Apply(matrix.TensorProduct(gate.X(), gate.I(4)))
	}

	if m3.IsOne() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(), gate.X(), gate.I(3)))
	}

	if m3.IsZero() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(2), gate.X(), gate.I(2)))
	}

	// answer is |111>|10>
	if phi.Probability()[30] != 1 {
		t.Error(phi.Probability())
	}
}

func TestErrorCorrectionBitFlip1(t *testing.T) {
	phi := qubit.New(1, 2)

	// encoding
	phi.TensorProduct(qubit.Zero(2))
	phi.Apply(gate.CNOT(3, 0, 1))
	phi.Apply(gate.CNOT(3, 0, 2))

	// error: first qubit is flipped
	phi.Apply(matrix.TensorProduct(gate.X(), gate.I(2)))

	// add ancilla qubit
	phi.TensorProduct(qubit.Zero(2))

	// z1z2
	phi.Apply(gate.CNOT(5, 0, 3)).Apply(gate.CNOT(5, 1, 3))

	// z2z3
	phi.Apply(gate.CNOT(5, 1, 4)).Apply(gate.CNOT(5, 2, 4))

	// measure
	m3 := phi.Measure(3)
	m4 := phi.Measure(4)

	// recover
	if m3.IsOne() && m4.IsZero() {
		phi.Apply(matrix.TensorProduct(gate.X(), gate.I(4)))
	}

	if m3.IsOne() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(), gate.X(), gate.I(3)))
	}

	if m3.IsZero() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(2), gate.X(), gate.I(2)))
	}

	// answer is 0.2|000>|10> + 0.8|111>|10>
	p := phi.Probability()
	if math.Abs(p[2]-0.2) > 1e-13 {
		t.Error(p)
	}

	if math.Abs(p[30]-0.8) > 1e-13 {
		t.Error(p)
	}
}

func TestErrorCorrectionBitFlip2(t *testing.T) {
	phi := qubit.New(1, 2)

	// encoding
	phi.TensorProduct(qubit.Zero(2))
	phi.Apply(gate.CNOT(3, 0, 1))
	phi.Apply(gate.CNOT(3, 0, 2))

	// error: second qubit is flipped
	phi.Apply(matrix.TensorProduct(gate.I(), gate.X(), gate.I()))

	// add ancilla qubit
	phi.TensorProduct(qubit.Zero(2))

	// z1z2
	phi.Apply(gate.CNOT(5, 0, 3)).Apply(gate.CNOT(5, 1, 3))

	// z2z3
	phi.Apply(gate.CNOT(5, 1, 4)).Apply(gate.CNOT(5, 2, 4))

	// measure
	m3 := phi.Measure(3)
	m4 := phi.Measure(4)

	// recover
	if m3.IsOne() && m4.IsZero() {
		phi.Apply(matrix.TensorProduct(gate.X(), gate.I(4)))
	}

	if m3.IsOne() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(), gate.X(), gate.I(3)))
	}

	if m3.IsZero() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(2), gate.X(), gate.I(2)))
	}

	// answer is 0.2|000>|11> + 0.8|111>|11>
	p := phi.Probability()
	if math.Abs(p[3]-0.2) > 1e-13 {
		t.Error(p)
	}

	if math.Abs(p[31]-0.8) > 1e-13 {
		t.Error(p)
	}
}

func TestErrorCorrectionBitFlip3(t *testing.T) {
	phi := qubit.New(1, 2)

	// encoding
	phi.TensorProduct(qubit.Zero(2))
	phi.Apply(gate.CNOT(3, 0, 1))
	phi.Apply(gate.CNOT(3, 0, 2))

	// error: third qubit is flipped
	phi.Apply(matrix.TensorProduct(gate.I(), gate.I(), gate.X()))

	// add ancilla qubit
	phi.TensorProduct(qubit.Zero(2))

	// z1z2
	phi.Apply(gate.CNOT(5, 0, 3)).Apply(gate.CNOT(5, 1, 3))

	// z2z3
	phi.Apply(gate.CNOT(5, 1, 4)).Apply(gate.CNOT(5, 2, 4))

	// measure
	m3 := phi.Measure(3)
	m4 := phi.Measure(4)

	// recover
	if m3.IsOne() && m4.IsZero() {
		phi.Apply(matrix.TensorProduct(gate.X(), gate.I(4)))
	}

	if m3.IsOne() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(), gate.X(), gate.I(3)))
	}

	if m3.IsZero() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(2), gate.X(), gate.I(2)))
	}

	// answer is 0.2|000>|01> + 0.8|111>|01>
	p := phi.Probability()
	if math.Abs(p[1]-0.2) > 1e-13 {
		t.Error(p)
	}

	if math.Abs(p[29]-0.8) > 1e-13 {
		t.Error(p)
	}
}

func TestErrorCorrectionPhaseFlip1(t *testing.T) {
	phi := qubit.New(1, 2)

	// encoding
	phi.TensorProduct(qubit.Zero(2))
	phi.Apply(gate.CNOT(3, 0, 1))
	phi.Apply(gate.CNOT(3, 0, 2))
	phi.Apply(gate.H(3))

	// error: first qubit is flipped
	phi.Apply(matrix.TensorProduct(gate.Z(), gate.I(2)))

	// H
	phi.Apply(gate.H(3))

	// add ancilla qubit
	phi.TensorProduct(qubit.Zero(2))

	// x1x2
	phi.Apply(gate.CNOT(5, 0, 3)).Apply(gate.CNOT(5, 1, 3))

	// x2x3
	phi.Apply(gate.CNOT(5, 1, 4)).Apply(gate.CNOT(5, 2, 4))

	// H
	phi.Apply(matrix.TensorProduct(gate.H(3), gate.I(2)))

	// measure
	m3 := phi.Measure(3)
	m4 := phi.Measure(4)

	// recover
	if m3.IsOne() && m4.IsZero() {
		phi.Apply(matrix.TensorProduct(gate.Z(), gate.I(4)))
	}

	if m3.IsOne() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(), gate.Z(), gate.I(3)))
	}

	if m3.IsZero() && m4.IsOne() {
		phi.Apply(matrix.TensorProduct(gate.I(2), gate.Z(), gate.I(2)))
	}

	phi.Apply(matrix.TensorProduct(gate.H(3), gate.I(2)))

	p := phi.Probability()
	if math.Abs(p[2]-0.2) > 1e-13 {
		t.Error(p)
	}

	if math.Abs(p[30]-0.8) > 1e-13 {
		t.Error(p)
	}
}
