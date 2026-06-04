package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type PyContractDate string
type PyContractDecimal string
type PyContractString string
type PyContractInt int

type PythonContractScalarPack struct {
	Name            string            `json:"name"`
	Enabled         bool              `json:"enabled"`
	Count           int               `json:"count"`
	SmallInt        int8              `json:"small_int"`
	MediumInt       int16             `json:"medium_int"`
	WideInt         int32             `json:"wide_int"`
	HugeInt         int64             `json:"huge_int"`
	Unsigned        uint              `json:"unsigned"`
	UnsignedSmall   uint8             `json:"unsigned_small"`
	UnsignedMedium  uint16            `json:"unsigned_medium"`
	UnsignedWide    uint32            `json:"unsigned_wide"`
	UnsignedHuge    uint64            `json:"unsigned_huge"`
	Ratio32         float32           `json:"ratio32"`
	Ratio64         float64           `json:"ratio64"`
	AliasString     PyContractString  `json:"alias_string"`
	AliasInt        PyContractInt     `json:"alias_int"`
	OptionalString  *string           `json:"optional_string,omitempty"`
	MissingOptional *string           `json:"missing_optional,omitempty"`
	EmptyStrings    []string          `json:"empty_strings"`
	Labels          map[string]string `json:"labels"`
	RawObject       json.RawMessage   `json:"raw_object"`
	RawList         json.RawMessage   `json:"raw_list"`
}

type PythonContractTemporalPack struct {
	UTCStamp        time.Time                 `json:"utc_stamp"`
	OffsetStamp     time.Time                 `json:"offset_stamp"`
	NegativeOffset  time.Time                 `json:"negative_offset"`
	NaiveStamp      time.Time                 `json:"naive_stamp"`
	FractionalStamp time.Time                 `json:"fractional_stamp"`
	Date            PyContractDate            `json:"date"`
	OptionalStamp   *time.Time                `json:"optional_stamp,omitempty"`
	OptionalDate    *PyContractDate           `json:"optional_date,omitempty"`
	StampList       []time.Time               `json:"stamp_list"`
	DateMap         map[string]PyContractDate `json:"date_map"`
}

type PythonContractDecimalPack struct {
	Money       PyContractDecimal            `json:"money"`
	MoneyNumber PyContractDecimal            `json:"money_number"`
	Negative    PyContractDecimal            `json:"negative"`
	Small       PyContractDecimal            `json:"small"`
	Large       PyContractDecimal            `json:"large"`
	Zero        PyContractDecimal            `json:"zero"`
	Optional    *PyContractDecimal           `json:"optional,omitempty"`
	Values      []PyContractDecimal          `json:"values"`
	Lookup      map[string]PyContractDecimal `json:"lookup"`
	FloatValue  float64                      `json:"float_value"`
}

type PythonContractNestedLeaf struct {
	ID     string            `json:"id"`
	Score  int               `json:"score"`
	When   time.Time         `json:"when"`
	Amount PyContractDecimal `json:"amount"`
	Date   PyContractDate    `json:"date"`
}

type PythonContractNestedNode struct {
	Leaf       PythonContractNestedLeaf              `json:"leaf"`
	MaybeLeaf  *PythonContractNestedLeaf             `json:"maybe_leaf,omitempty"`
	Leaves     []PythonContractNestedLeaf            `json:"leaves"`
	LeafMap    map[string]PythonContractNestedLeaf   `json:"leaf_map"`
	LeafGroups map[string][]PythonContractNestedLeaf `json:"leaf_groups"`
	Matrix     [][]PythonContractNestedLeaf          `json:"matrix"`
	AnyPayload json.RawMessage                       `json:"any_payload"`
}

type PythonContractWirePack struct {
	From         string `json:"from"`
	Class        string `json:"class"`
	Try          string `json:"try"`
	Else         string `json:"else"`
	List         string `json:"list"`
	Dict         string `json:"dict"`
	Type         string `json:"type"`
	Object       string `json:"object"`
	ID           string `json:"id"`
	DecodeValue  string `json:"_decode_value"`
	CreateClient string `json:"create_client"`
	FromLiteral  string `json:"from_"`
	Date         string `json:"date"`
}

type PythonContractRequest struct {
	Scalar   PythonContractScalarPack   `json:"scalar"`
	Temporal PythonContractTemporalPack `json:"temporal"`
	Decimal  PythonContractDecimalPack  `json:"decimal"`
	Nested   PythonContractNestedNode   `json:"nested"`
	Wire     PythonContractWirePack     `json:"wire"`
}

type PythonContractResponse struct {
	Scalar   PythonContractScalarPack   `json:"scalar"`
	Temporal PythonContractTemporalPack `json:"temporal"`
	Decimal  PythonContractDecimalPack  `json:"decimal"`
	Nested   PythonContractNestedNode   `json:"nested"`
	Wire     PythonContractWirePack     `json:"wire"`
}

func pythonMegaContractHandler(ctx context.Context, req PythonContractRequest) (PythonContractResponse, int) {
	_ = ctx
	_ = req
	return PythonContractResponse{}, StatusOK
}

func TestRPCPythonMegaContractDecodingAndEncoding(t *testing.T) {
	router := NewRouter()
	router.SetTypeOverrides(map[string]TypeOverride{
		"PyContractDate": {
			JSType:        "string",
			PyType:        "date",
			OpenAPIType:   "string",
			OpenAPIFormat: "date",
		},
		"PyContractDecimal": {
			JSType:        "string",
			PyType:        "Decimal",
			OpenAPIType:   "string",
			OpenAPIFormat: "decimal",
		},
	})
	router.HandleRPC(pythonMegaContractHandler)

	var buf bytes.Buffer
	if err := router.WriteClientPY(&buf); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	py := buf.Bytes()
	pyText := string(py)
	assertRPCContains(t, pyText, "from datetime import date as _date, datetime as _datetime")
	assertRPCContains(t, pyText, "from decimal import Decimal as _Decimal")
	assertRPCContains(t, pyText, "utc_stamp: _datetime")
	assertRPCContains(t, pyText, "date: _date")
	assertRPCContains(t, pyText, "money: _Decimal")

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runRPCPython("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}

	snippet := pythonRPCImportSnippet(pyPath) + `
from dataclasses import is_dataclass
from datetime import date, datetime, timezone, timedelta
from decimal import Decimal
import json

request_payload = {
    "scalar": {
        "name": "request",
        "enabled": True,
        "count": -7,
        "small_int": -8,
        "medium_int": -1600,
        "wide_int": -320000,
        "huge_int": -9007199254740991,
        "unsigned": 7,
        "unsigned_small": 8,
        "unsigned_medium": 1600,
        "unsigned_wide": 320000,
        "unsigned_huge": 9007199254740991,
        "ratio32": 1.5,
        "ratio64": -2.75,
        "alias_string": "alias",
        "alias_int": 42,
        "optional_string": "present",
        "empty_strings": [],
        "labels": {"region": "west", "tier": "gold"},
        "raw_object": {"ok": True, "count": 2},
        "raw_list": [{"id": "raw-1"}, 3, False],
    },
    "temporal": {
        "utc_stamp": "2025-01-02T03:04:05Z",
        "offset_stamp": "2025-01-02T03:04:05+05:30",
        "negative_offset": "2025-01-02T03:04:05-07:00",
        "naive_stamp": "2025-01-02T03:04:05",
        "fractional_stamp": "2025-01-02T03:04:05.123456Z",
        "date": "2025-03-04",
        "optional_stamp": "2025-01-03T04:05:06Z",
        "optional_date": "2025-03-05",
        "stamp_list": ["2025-01-04T00:00:00Z", "2025-01-05T12:30:45+02:00"],
        "date_map": {"start": "2025-04-01", "end": "2025-04-30"},
    },
    "decimal": {
        "money": "123.4500",
        "money_number": 99.25,
        "negative": "-12.3400",
        "small": "0.000000000000000001",
        "large": "12345678901234567890.123456789",
        "zero": "0",
        "optional": "42.000",
        "values": ["1.10", "2.20", "3.30"],
        "lookup": {"tax": "8.875", "discount": "-1.005"},
        "float_value": 88.125,
    },
    "nested": {
        "leaf": {"id": "leaf-1", "score": 10, "when": "2025-02-01T01:02:03Z", "amount": "10.50", "date": "2025-02-01"},
        "maybe_leaf": {"id": "leaf-2", "score": 11, "when": "2025-02-02T01:02:03Z", "amount": "11.50", "date": "2025-02-02"},
        "leaves": [
            {"id": "leaf-3", "score": 12, "when": "2025-02-03T01:02:03Z", "amount": "12.50", "date": "2025-02-03"},
            {"id": "leaf-4", "score": 13, "when": "2025-02-04T01:02:03Z", "amount": "13.50", "date": "2025-02-04"},
        ],
        "leaf_map": {
            "primary": {"id": "leaf-5", "score": 14, "when": "2025-02-05T01:02:03Z", "amount": "14.50", "date": "2025-02-05"}
        },
        "leaf_groups": {
            "batch": [{"id": "leaf-6", "score": 15, "when": "2025-02-06T01:02:03Z", "amount": "15.50", "date": "2025-02-06"}]
        },
        "matrix": [[{"id": "leaf-7", "score": 16, "when": "2025-02-07T01:02:03Z", "amount": "16.50", "date": "2025-02-07"}]],
        "any_payload": {"mixed": [1, "two", True, None]},
    },
    "wire": {
        "from": "source",
        "class": "tier",
        "try": "attempt",
        "else": "fallback",
        "list": "list-value",
        "dict": "dict-value",
        "type": "type-value",
        "object": "object-value",
        "id": "id-value",
        "_decode_value": "decode-value",
        "create_client": "factory-value",
        "from_": "literal-from-underscore",
        "date": "wire-date",
    },
}

response_payload = json.loads(json.dumps(request_payload))
response_payload["scalar"]["name"] = "response"

body = mod._decode_value(mod.PythonContractRequest, request_payload)
assert is_dataclass(body)
assert isinstance(body.scalar, mod.PythonContractScalarPack)
assert isinstance(body.temporal, mod.PythonContractTemporalPack)
assert isinstance(body.decimal, mod.PythonContractDecimalPack)
assert isinstance(body.nested, mod.PythonContractNestedNode)
assert isinstance(body.wire, mod.PythonContractWirePack)

assert body.scalar.name == "request"
assert body.scalar.enabled is True
assert body.scalar.count == -7
assert body.scalar.small_int == -8
assert body.scalar.medium_int == -1600
assert body.scalar.wide_int == -320000
assert body.scalar.huge_int == -9007199254740991
assert body.scalar.unsigned == 7
assert body.scalar.unsigned_small == 8
assert body.scalar.unsigned_medium == 1600
assert body.scalar.unsigned_wide == 320000
assert body.scalar.unsigned_huge == 9007199254740991
assert isinstance(body.scalar.ratio32, float)
assert isinstance(body.scalar.ratio64, float)
assert body.scalar.alias_string == "alias"
assert body.scalar.alias_int == 42
assert body.scalar.optional_string == "present"
assert body.scalar.missing_optional is None
assert body.scalar.empty_strings == []
assert body.scalar.labels["tier"] == "gold"
assert body.scalar.raw_object["ok"] is True
assert body.scalar.raw_list[0]["id"] == "raw-1"

assert isinstance(body.temporal.utc_stamp, datetime)
assert body.temporal.utc_stamp.tzinfo is not None
assert body.temporal.utc_stamp.utcoffset() == timedelta(0)
assert body.temporal.offset_stamp.utcoffset() == timedelta(hours=5, minutes=30)
assert body.temporal.negative_offset.utcoffset() == -timedelta(hours=7)
assert body.temporal.naive_stamp.tzinfo is None
assert body.temporal.fractional_stamp.microsecond == 123456
assert isinstance(body.temporal.date, date) and not isinstance(body.temporal.date, datetime)
assert body.temporal.date.isoformat() == "2025-03-04"
assert body.temporal.optional_stamp.year == 2025
assert body.temporal.optional_date.isoformat() == "2025-03-05"
assert len(body.temporal.stamp_list) == 2
assert body.temporal.stamp_list[1].utcoffset() == timedelta(hours=2)
assert body.temporal.date_map["end"].isoformat() == "2025-04-30"

assert isinstance(body.decimal.money, Decimal)
assert body.decimal.money == Decimal("123.4500")
assert body.decimal.money_number == Decimal("99.25")
assert body.decimal.negative == Decimal("-12.3400")
assert body.decimal.small == Decimal("0.000000000000000001")
assert body.decimal.large == Decimal("12345678901234567890.123456789")
assert body.decimal.zero == Decimal("0")
assert body.decimal.optional == Decimal("42.000")
assert body.decimal.values == [Decimal("1.10"), Decimal("2.20"), Decimal("3.30")]
assert body.decimal.lookup["discount"] == Decimal("-1.005")
assert isinstance(body.decimal.float_value, float)

assert isinstance(body.nested.leaf, mod.PythonContractNestedLeaf)
assert isinstance(body.nested.maybe_leaf, mod.PythonContractNestedLeaf)
assert isinstance(body.nested.leaves[0], mod.PythonContractNestedLeaf)
assert isinstance(body.nested.leaf_map["primary"], mod.PythonContractNestedLeaf)
assert isinstance(body.nested.leaf_groups["batch"][0], mod.PythonContractNestedLeaf)
assert isinstance(body.nested.matrix[0][0], mod.PythonContractNestedLeaf)
assert body.nested.leaf.when.utcoffset() == timedelta(0)
assert body.nested.leaf.amount == Decimal("10.50")
assert body.nested.leaf.date.isoformat() == "2025-02-01"
assert body.nested.any_payload["mixed"][3] is None

assert body.wire.from_ == "source"
assert body.wire.class_ == "tier"
assert body.wire.try_ == "attempt"
assert body.wire.else_ == "fallback"
assert body.wire.list == "list-value"
assert body.wire.dict == "dict-value"
assert body.wire.type_ == "type-value"
assert body.wire.object == "object-value"
assert body.wire.id == "id-value"
assert body.wire._decode_value == "decode-value"
assert body.wire.create_client == "factory-value"
assert body.wire.from_2 == "literal-from-underscore"
assert body.wire.date == "wire-date"

encoded = mod._encode_value(body)
assert encoded["decimal"]["money"] == "123.4500"
assert encoded["decimal"]["money_number"] == "99.25"
assert encoded["temporal"]["date"] == "2025-03-04"
assert encoded["temporal"]["utc_stamp"].startswith("2025-01-02T03:04:05")
assert encoded["wire"]["from"] == "source"
assert encoded["wire"]["class"] == "tier"
assert encoded["wire"]["id"] == "id-value"
assert encoded["wire"]["create_client"] == "factory-value"

calls = []
statuses = [200, 422, 500]

class FakeResponse:
    def __init__(self, status, payload):
        self._status = status
        self._payload = payload
    def __enter__(self):
        return self
    def __exit__(self, exc_type, exc, tb):
        return False
    def getcode(self):
        return self._status
    def read(self):
        return self._payload

def fake_urlopen(req):
    sent = json.loads(req.data.decode("utf-8"))
    calls.append((req, sent))
    status = statuses.pop(0)
    if status == 500:
        return FakeResponse(status, b"not json")
    return FakeResponse(status, json.dumps(response_payload).encode("utf-8"))

mod.request.urlopen = fake_urlopen
client = mod.create_client(base_url="https://core.example")

resp = client.rpc.pythonMegaContractHandler(body)
assert isinstance(resp, mod.PythonContractResponse)
assert isinstance(resp.scalar, mod.PythonContractScalarPack)
assert resp.scalar.name == "response"
assert isinstance(resp.temporal.utc_stamp, datetime)
assert resp.temporal.offset_stamp.utcoffset() == timedelta(hours=5, minutes=30)
assert isinstance(resp.temporal.date, date) and not isinstance(resp.temporal.date, datetime)
assert resp.decimal.money == Decimal("123.4500")
assert isinstance(resp.nested.leaves[1], mod.PythonContractNestedLeaf)
assert resp.nested.matrix[0][0].amount == Decimal("16.50")
assert resp.wire.create_client == "factory-value"

sent = calls[0][1]
assert sent["decimal"]["money"] == "123.4500"
assert sent["decimal"]["small"] == "0.000000000000000001"
assert sent["temporal"]["date"] == "2025-03-04"
assert sent["nested"]["leaf"]["amount"] == "10.50"
assert sent["wire"]["from"] == "source"
assert sent["wire"]["from_"] == "literal-from-underscore"

try:
    client.rpc.pythonMegaContractHandler(body)
    raise AssertionError("expected RPCError for 422")
except mod.RPCError as err:
    assert err.status == 422
    assert isinstance(err.body, mod.PythonContractResponse)
    assert err.body.decimal.money == Decimal("123.4500")
    assert isinstance(err.body.temporal.utc_stamp, datetime)
    assert err.body.nested.leaf.date.isoformat() == "2025-02-01"

try:
    client.rpc.pythonMegaContractHandler(body)
    raise AssertionError("expected RPCError for non-json 500")
except mod.RPCError as err:
    assert err.status == 500
    assert err.body is None
`
	if err := runRPCPython("-c", snippet); err != nil {
		t.Fatalf("python mega contract failed: %v", err)
	}
}
