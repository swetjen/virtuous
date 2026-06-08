package httpapi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type HTTPPyContractDate string
type HTTPPyContractDecimal string
type HTTPPyContractString string
type HTTPPyContractInt int

type HTTPPythonContractScalarPack struct {
	Name            string               `json:"name"`
	Enabled         bool                 `json:"enabled"`
	Count           int                  `json:"count"`
	SmallInt        int8                 `json:"small_int"`
	MediumInt       int16                `json:"medium_int"`
	WideInt         int32                `json:"wide_int"`
	HugeInt         int64                `json:"huge_int"`
	Unsigned        uint                 `json:"unsigned"`
	UnsignedSmall   uint8                `json:"unsigned_small"`
	UnsignedMedium  uint16               `json:"unsigned_medium"`
	UnsignedWide    uint32               `json:"unsigned_wide"`
	UnsignedHuge    uint64               `json:"unsigned_huge"`
	Ratio32         float32              `json:"ratio32"`
	Ratio64         float64              `json:"ratio64"`
	AliasString     HTTPPyContractString `json:"alias_string"`
	AliasInt        HTTPPyContractInt    `json:"alias_int"`
	OptionalString  *string              `json:"optional_string,omitempty"`
	MissingOptional *string              `json:"missing_optional,omitempty"`
	EmptyStrings    []string             `json:"empty_strings"`
	Labels          map[string]string    `json:"labels"`
	RawObject       json.RawMessage      `json:"raw_object"`
	RawList         json.RawMessage      `json:"raw_list"`
}

type HTTPPythonContractTemporalPack struct {
	UTCStamp        time.Time                     `json:"utc_stamp"`
	OffsetStamp     time.Time                     `json:"offset_stamp"`
	NegativeOffset  time.Time                     `json:"negative_offset"`
	NaiveStamp      time.Time                     `json:"naive_stamp"`
	FractionalStamp time.Time                     `json:"fractional_stamp"`
	Date            HTTPPyContractDate            `json:"date"`
	OptionalStamp   *time.Time                    `json:"optional_stamp,omitempty"`
	OptionalDate    *HTTPPyContractDate           `json:"optional_date,omitempty"`
	StampList       []time.Time                   `json:"stamp_list"`
	DateMap         map[string]HTTPPyContractDate `json:"date_map"`
}

type HTTPPythonContractDecimalPack struct {
	Money       HTTPPyContractDecimal            `json:"money"`
	MoneyNumber HTTPPyContractDecimal            `json:"money_number"`
	Negative    HTTPPyContractDecimal            `json:"negative"`
	Small       HTTPPyContractDecimal            `json:"small"`
	Large       HTTPPyContractDecimal            `json:"large"`
	Zero        HTTPPyContractDecimal            `json:"zero"`
	Optional    *HTTPPyContractDecimal           `json:"optional,omitempty"`
	Values      []HTTPPyContractDecimal          `json:"values"`
	Lookup      map[string]HTTPPyContractDecimal `json:"lookup"`
	FloatValue  float64                          `json:"float_value"`
}

type HTTPPythonContractNestedLeaf struct {
	ID     string                `json:"id"`
	Score  int                   `json:"score"`
	When   time.Time             `json:"when"`
	Amount HTTPPyContractDecimal `json:"amount"`
	Date   HTTPPyContractDate    `json:"date"`
}

type HTTPPythonContractNestedNode struct {
	Leaf       HTTPPythonContractNestedLeaf              `json:"leaf"`
	MaybeLeaf  *HTTPPythonContractNestedLeaf             `json:"maybe_leaf,omitempty"`
	Leaves     []HTTPPythonContractNestedLeaf            `json:"leaves"`
	LeafMap    map[string]HTTPPythonContractNestedLeaf   `json:"leaf_map"`
	LeafGroups map[string][]HTTPPythonContractNestedLeaf `json:"leaf_groups"`
	Matrix     [][]HTTPPythonContractNestedLeaf          `json:"matrix"`
	AnyPayload json.RawMessage                           `json:"any_payload"`
}

type HTTPPythonContractWirePack struct {
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

type HTTPPythonMegaRequest struct {
	Scalar   HTTPPythonContractScalarPack   `json:"scalar"`
	Temporal HTTPPythonContractTemporalPack `json:"temporal"`
	Decimal  HTTPPythonContractDecimalPack  `json:"decimal"`
	Nested   HTTPPythonContractNestedNode   `json:"nested"`
	Wire     HTTPPythonContractWirePack     `json:"wire"`
}

type HTTPPythonMegaResponse struct {
	Scalar   HTTPPythonContractScalarPack   `json:"scalar"`
	Temporal HTTPPythonContractTemporalPack `json:"temporal"`
	Decimal  HTTPPythonContractDecimalPack  `json:"decimal"`
	Nested   HTTPPythonContractNestedNode   `json:"nested"`
	Wire     HTTPPythonContractWirePack     `json:"wire"`
}

type HTTPPythonSearchRequest struct {
	AccountID string   `path:"account_id"`
	IDs       []string `query:"id"`
	Limit     int      `query:"limit"`
	Filter    string   `query:"filter,omitempty"`
}

type HTTPPythonSearchResponse struct {
	Count int `json:"count"`
}

type HTTPPythonOptionalBody struct {
	Name   string                `json:"name"`
	Amount HTTPPyContractDecimal `json:"amount"`
}

type HTTPPythonOptionalResponse struct {
	Accepted bool `json:"accepted"`
}

type HTTPPythonNoBodyRequest struct {
	AccountID string `path:"account_id"`
}

type HTTPPythonMixedRequest struct {
	AccountID string                `path:"account_id"`
	IDs       []string              `query:"id"`
	Limit     int                   `query:"limit,omitempty"`
	Name      string                `json:"name"`
	Amount    HTTPPyContractDecimal `json:"amount"`
	At        time.Time             `json:"at"`
}

func TestHTTPAPIPythonMegaContractDecodingEncodingAndTransport(t *testing.T) {
	router := NewRouter()
	router.SetTypeOverrides(httpPythonContractTypeOverrides())
	router.Describe("POST /contracts/python/mega", HTTPPythonMegaRequest{}, HTTPPythonMegaResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Mega",
		OperationID: "python_mega",
	})
	router.Describe("GET /contracts/{account_id}/search", HTTPPythonSearchRequest{}, HTTPPythonSearchResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Search",
		OperationID: "python_search",
	})
	router.Describe("PATCH /contracts/python/optional", Optional[HTTPPythonOptionalBody](), HTTPPythonOptionalResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Optional",
		OperationID: "python_optional",
	})
	router.Describe("DELETE /contracts/{account_id}/cache", HTTPPythonNoBodyRequest{}, NoResponse204{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "ClearCache",
		OperationID: "python_clear_cache",
	})
	router.Describe("PUT /contracts/{account_id}/mixed", HTTPPythonMixedRequest{}, HTTPPythonOptionalResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Mixed",
		OperationID: "python_mixed",
	})

	var buf bytes.Buffer
	if err := router.WriteClientPY(&buf); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	py := buf.Bytes()
	pyText := string(py)
	assertContains(t, pyText, "from datetime import date as _date, datetime as _datetime")
	assertContains(t, pyText, "from decimal import Decimal as _Decimal")
	assertContains(t, pyText, "utc_stamp: _datetime")
	assertContains(t, pyText, "date: _date")
	assertContains(t, pyText, "money: _Decimal")
	assertContains(t, pyText, "raw_object: Any")
	assertContains(t, pyText, "def python_search(self, account_id: str, *, id: list[str], limit: int, filter: Optional[str] = None)")
	assertContains(t, pyText, `def python_optional(self, *, body: Optional["ContractsHTTPPythonOptionalBody"] = None)`)
	assertContains(t, pyText, "def python_clear_cache(self, account_id: str) -> None")
	assertContains(t, pyText, `def python_mixed(self, account_id: str, *, id: list[str], body: Optional["ContractsHTTPPythonMixedRequest"] = None, limit: Optional[int] = None)`)

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runPythonCommand("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}

	snippet := pythonImportSnippet(pyPath) + `
from dataclasses import is_dataclass
from datetime import date, datetime, timedelta
from decimal import Decimal
from urllib import parse as urlparse
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

body = mod._decode_value(mod.ContractsHTTPPythonMegaRequest, request_payload)
assert is_dataclass(body)
assert isinstance(body.scalar, mod.HTTPPythonContractScalarPack)
assert isinstance(body.temporal, mod.HTTPPythonContractTemporalPack)
assert isinstance(body.decimal, mod.HTTPPythonContractDecimalPack)
assert isinstance(body.nested, mod.HTTPPythonContractNestedNode)
assert isinstance(body.wire, mod.HTTPPythonContractWirePack)

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

assert isinstance(body.nested.leaf, mod.HTTPPythonContractNestedLeaf)
assert isinstance(body.nested.maybe_leaf, mod.HTTPPythonContractNestedLeaf)
assert isinstance(body.nested.leaves[0], mod.HTTPPythonContractNestedLeaf)
assert isinstance(body.nested.leaf_map["primary"], mod.HTTPPythonContractNestedLeaf)
assert isinstance(body.nested.leaf_groups["batch"][0], mod.HTTPPythonContractNestedLeaf)
assert isinstance(body.nested.matrix[0][0], mod.HTTPPythonContractNestedLeaf)
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
statuses = [200, 422, 204]

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
    calls.append(req)
    if req.full_url.endswith("/contracts/python/mega"):
        sent = json.loads(req.data.decode("utf-8"))
        assert req.get_method() == "POST"
        headers = {key.lower(): value for key, value in req.header_items()}
        assert headers["accept"] == "application/json"
        assert headers["content-type"] == "application/json"
        assert sent["decimal"]["money"] == "123.4500"
        assert sent["decimal"]["small"] == "0.000000000000000001"
        assert sent["temporal"]["date"] == "2025-03-04"
        assert sent["nested"]["leaf"]["amount"] == "10.50"
        assert sent["wire"]["from"] == "source"
        assert sent["wire"]["from_"] == "literal-from-underscore"
        status = statuses.pop(0)
        if status == 422:
            return FakeResponse(status, b'{"error":"bad mega"}')
        return FakeResponse(status, json.dumps(response_payload).encode("utf-8"))
    if "/contracts/acct%201/search" in req.full_url:
        assert req.get_method() == "GET"
        assert req.data is None
        parts = urlparse.urlsplit(req.full_url)
        assert parts.scheme + "://" + parts.netloc + parts.path == "https://core.example/contracts/acct%201/search"
        assert urlparse.parse_qs(parts.query) == {"id": ["a", "b"], "limit": ["50"], "filter": ["hot"]}
        return FakeResponse(200, b'{"count":2}')
    if req.full_url.endswith("/contracts/python/optional"):
        assert req.get_method() == "PATCH"
        if req.data is None:
            return FakeResponse(200, b'{"accepted":false}')
        sent = json.loads(req.data.decode("utf-8"))
        assert sent == {"name": "optional", "amount": "44.10"}
        return FakeResponse(200, b'{"accepted":true}')
    if "/contracts/acct-2/cache" in req.full_url:
        assert req.get_method() == "DELETE"
        assert req.data is None
        return FakeResponse(204, b"")
    if "/contracts/acct%203/mixed" in req.full_url:
        assert req.get_method() == "PUT"
        parts = urlparse.urlsplit(req.full_url)
        assert urlparse.parse_qs(parts.query) == {"id": ["x", "y"], "limit": ["25"]}
        sent = json.loads(req.data.decode("utf-8"))
        assert sent == {"name": "mixed", "amount": "12.3400", "at": "2025-07-01T02:03:04+00:00"}, sent
        return FakeResponse(200, b'{"accepted":true}')
    raise AssertionError("unexpected request " + req.full_url)

mod.request.urlopen = fake_urlopen
client = mod.create_client(base_url="https://core.example")

resp = client.python_mega(body=body)
assert isinstance(resp, mod.ContractsHTTPPythonMegaResponse)
assert isinstance(resp.scalar, mod.HTTPPythonContractScalarPack)
assert resp.scalar.name == "response"
assert isinstance(resp.temporal.utc_stamp, datetime)
assert resp.temporal.offset_stamp.utcoffset() == timedelta(hours=5, minutes=30)
assert isinstance(resp.temporal.date, date) and not isinstance(resp.temporal.date, datetime)
assert resp.decimal.money == Decimal("123.4500")
assert isinstance(resp.nested.leaves[1], mod.HTTPPythonContractNestedLeaf)
assert resp.nested.matrix[0][0].amount == Decimal("16.50")
assert resp.wire.create_client == "factory-value"

try:
    client.python_mega(body=body)
    raise AssertionError("expected RuntimeError for 422")
except RuntimeError as err:
    assert str(err) == "bad mega"

search = client.python_search("acct 1", id=["a", "b"], limit=50, filter="hot")
assert isinstance(search, mod.ContractsHTTPPythonSearchResponse)
assert search.count == 2

accepted = client.python_optional(body=mod.ContractsHTTPPythonOptionalBody(name="optional", amount=Decimal("44.10")))
assert accepted.accepted is True
empty = client.python_optional()
assert empty.accepted is False

assert client.python_clear_cache("acct-2") is None

mixed_kwargs = {}
for field in mod.fields(mod.ContractsHTTPPythonMixedRequest):
    wire = field.metadata.get("wire", field.name)
    if wire in ("AccountID", "accountID", "account_id"):
        mixed_kwargs[field.name] = "body-should-not-send"
    elif wire in ("IDs", "iDs", "ids", "id"):
        mixed_kwargs[field.name] = ["body-should-not-send"]
    elif wire in ("Limit", "limit"):
        mixed_kwargs[field.name] = 99
    elif wire == "name":
        mixed_kwargs[field.name] = "mixed"
    elif wire == "amount":
        mixed_kwargs[field.name] = Decimal("12.3400")
    elif wire == "at":
        mixed_kwargs[field.name] = datetime.fromisoformat("2025-07-01T02:03:04+00:00")
mixed = mod.ContractsHTTPPythonMixedRequest(**mixed_kwargs)
mixed_resp = client.python_mixed("acct 3", id=["x", "y"], body=mixed, limit=25)
assert mixed_resp.accepted is True
`
	if err := runPythonCommand("-c", snippet); err != nil {
		t.Fatalf("python httpapi mega contract failed: %v", err)
	}
}

func httpPythonContractTypeOverrides() map[string]TypeOverride {
	return map[string]TypeOverride{
		"HTTPPyContractDate": {
			JSType:        "string",
			PyType:        "date",
			OpenAPIType:   "string",
			OpenAPIFormat: "date",
		},
		"HTTPPyContractDecimal": {
			JSType:        "string",
			PyType:        "Decimal",
			OpenAPIType:   "string",
			OpenAPIFormat: "decimal",
		},
	}
}
