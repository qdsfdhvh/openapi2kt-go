# Checked String enums and collection defaults

Use these roles only when the shared reference/model planner already admits the
source. A default annotation never grants a new reference or graph shape.
Enable `--validation` for supported String length/pattern and container checks.
String enum codecs keep their lexical `serialName` and open `Unrecognized`
values, including when format/native metadata is present. Such metadata creates
no date, UUID or provider codec for a model String enum.

## Nested inline bounds

Ordinary DTO direct inline finite List/typed Map trees may end in bounded String
enum leaves. Their own outer field defaults and already admitted object-default
walks validate every supplied leaf against its original vocabulary and checks.
Check missing/null, required/Patch, each container's nullable/count rules and
existing structural equality. Child-owned annotations and unadmitted composed
or recursive graphs keep their separate source restrictions.

## Collection roots

A finite List/typed Map root with an original inline lexical enum leaf retains
its actual collection alias and canonical enum peer. The root and pure named
aliases may each own independent default factories; accepted field/object
consumers retain their own annotations. Use the canonical enum owner for nested
singleton access through aliases. Typed DTO/HTTP/SSE consumers impose retained
serialName checks; standalone alias assignment and decoding remain open.

## Checked references

Existing named/internal references and borrowed collections may retain original
ordinary field or collection-root String checks and the actual lexical enum
classifier. Verify the enforcing source getter and consuming typed getter,
nearest nullable intent, required/Patch behavior and strict independent
annotations. Standalone enum/alias codecs remain open. Replacing a source,
classifier, backing or check object is not authority for a new source role.

## Borrowed collection defaults

An original ordinary List/typed Map field may keep its own annotation while an
actual internal collection alias and its finite pure aliases own independent
factories. Validate each annotation against its own original vocabulary,
checks and nullable view. Factories decode fresh container state through the
actual final serializer. Unannotated consumers inherit no value; child
annotations and general graphs remain separate roles.

## Checked enum roots

A bounded lexical String enum component and named aliases keep actual enum
classifiers and shared checks through eligible typed scalar, List/Map,
object-default, HTTP and SSE consumers. Root/alias/null factories are explicit
opt-in functions; their annotations require original vocabulary membership and
checks even though runtime codecs remain open. Nearest explicit nullable intent
wins, and required/Patch annotations validate before initializer suppression.
A factory remains tied to its actual root/export even without a consumer.

## Structural equality

With validation, eligible `uniqueItems:true` arrays of finite typed List/Map
trees compare mapped values recursively. List order matters; Map insertion
order does not. Null/empty containers, missing keys and null-valued entries stay
distinct. String enums compare serialName, including known/Unrecognized pairs;
floating values compare after mapping and normalize signed zero. Existing
UUID/date/time and checked native enum leaves retain their mapped equality and
source admission. These are typed Kotlin values, not general JSON Schema token
equality. Defaults use the same rule and report the second duplicate's original
Pointer. DTO/Patch.Value/typed HTTP/SSE boundaries enforce admitted checks;
standalone alias codecs and later caller mutation remain transparent. Raw,
DTO/union/hierarchy/provider/byte projections retain their separate roles.

## Verification and limits

Run `validate`, `generate` and `generate --check` with the same input/options.
Compile the fresh consumer and exercise the actual factories, missing/null
settings, construction/decode checks and relevant HTTP/SSE behavior. Report
source preparation separately from actual compilation/runtime. Existing
source48/work16,384/value16,384/default32KiB/checked16/enum240/JVM240 limits apply;
individual probes do not prove simultaneous maxima or another target platform.

Collections with reference leaves and independent component factories use the
[enum-reference collection guidance](referenced-enum-collection-factories.md).
These bounded roles do not imply complete default graphs, platform matrices or
release acceptance.
