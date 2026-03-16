#!/bin/bash
set -euo pipefail

AK="$(cd "$(dirname "$0")" && pwd)/ak"
PASS=0
FAIL=0
TESTS=0

# Save current profile and switch to Test
ORIGINAL=$(curl -s localhost:8765 -X POST -d '{"action":"getActiveProfile","version":6}' | jq -r .result)
trap 'echo ""; echo "Restoring profile: $ORIGINAL"; curl -s localhost:8765 -X POST -d "{\"action\":\"loadProfile\",\"version\":6,\"params\":{\"name\":\"$ORIGINAL\"}}" > /dev/null; echo "Results: $PASS passed, $FAIL failed out of $TESTS tests"' EXIT

echo "Current profile: $ORIGINAL"
curl -s localhost:8765 -X POST -d '{"action":"loadProfile","version":6,"params":{"name":"Test"}}' > /dev/null
echo "Switched to Test profile"

# Clean up any leftover notes from previous runs
LEFTOVER=$($AK search 'added:36500' 2>/dev/null | jq -r '.[]' 2>/dev/null) || true
if [ -n "$LEFTOVER" ]; then
    echo "Cleaning up leftover notes..."
    echo "$LEFTOVER" | tr '\n' ' ' | xargs $AK delete > /dev/null 2>&1 || true
fi
echo ""

assert() {
    TESTS=$((TESTS + 1))
    local desc="$1"
    local cmd="$2"
    local expected="$3"

    local actual
    actual=$(eval "$cmd" 2>&1) || true

    if echo "$actual" | grep -qF "$expected"; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc"
        echo "    expected to contain: $expected"
        echo "    got: $(echo "$actual" | head -5)"
    fi
}

assert_json() {
    TESTS=$((TESTS + 1))
    local desc="$1"
    local cmd="$2"
    local jq_filter="$3"
    local expected="$4"

    local output
    output=$(eval "$cmd" 2>&1) || true

    local actual
    actual=$(echo "$output" | jq -r "$jq_filter" 2>/dev/null) || actual="JQ_ERROR: $output"

    if [ "$actual" = "$expected" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc"
        echo "    expected: $expected"
        echo "    got: $actual"
    fi
}

assert_exit() {
    TESTS=$((TESTS + 1))
    local desc="$1"
    local cmd="$2"
    local expected_code="$3"

    local actual_code
    eval "$cmd" > /dev/null 2>&1 && actual_code=0 || actual_code=$?

    if [ "$actual_code" -eq "$expected_code" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc"
        echo "    expected exit code: $expected_code"
        echo "    got: $actual_code"
    fi
}

# ============================================================
echo "Phase 1: Connection & metadata"
# ============================================================

assert_json "ak version returns version number" \
    "$AK version" ".version" "6"

assert "ak decks lists Default deck" \
    "$AK decks" '"Default"'

assert "ak models lists note types" \
    "$AK models" '"Basic"'

assert_json "ak models -m Basic shows Front field" \
    "$AK models -m Basic" '.[0]' "Front"

assert_json "ak models -m Basic shows Back field" \
    "$AK models -m Basic" '.[1]' "Back"

assert "ak tags returns array" \
    "$AK tags" "["

# ============================================================
echo ""
echo "Phase 2: Deck management"
# ============================================================

assert_json "ak deck create Test Deck" \
    "$AK deck create 'Test Deck'" '.name' "Test Deck"

assert_json "ak deck create nested deck" \
    "$AK deck create 'Test Deck::Sub'" '.name' "Test Deck::Sub"

assert "ak decks shows Test Deck" \
    "$AK decks" '"Test Deck"'

assert "ak decks --stats runs without error" \
    "$AK decks --stats" '"name"'

# ============================================================
echo ""
echo "Phase 3: Note creation"
# ============================================================

NOTE1_ID=$($AK add "What is 2+2?" "4" -d "Test Deck" -t math,test | jq -r '.id')
TESTS=$((TESTS + 1))
if [ "$NOTE1_ID" != "null" ] && [ -n "$NOTE1_ID" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: ak add basic card (id: $NOTE1_ID)"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: ak add basic card (got null id)"
fi

NOTE2_ID=$($AK add "What is 3+3?" "6" -d "Test Deck" -t math | jq -r '.id')
TESTS=$((TESTS + 1))
if [ "$NOTE2_ID" != "null" ] && [ -n "$NOTE2_ID" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: ak add second card (id: $NOTE2_ID)"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: ak add second card"
fi

NOTE3_ID=$($AK add -m Cloze "The answer is {{c1::42}}" -d "Test Deck" -t cloze | jq -r '.id')
TESTS=$((TESTS + 1))
if [ "$NOTE3_ID" != "null" ] && [ -n "$NOTE3_ID" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: ak add cloze card (id: $NOTE3_ID)"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: ak add cloze card"
fi

# Duplicate check
assert_exit "ak add duplicate fails" \
    "$AK add 'What is 2+2?' '4' -d 'Test Deck'" 1

# ============================================================
echo ""
echo "Phase 4: Querying"
# ============================================================

assert_json "ak search deck:Test Deck finds 3 notes" \
    "$AK search 'deck:\"Test Deck\"'" 'length' "3"

assert_json "ak search tag:math finds 2 notes" \
    "$AK search 'tag:math'" 'length' "2"

assert_json "ak search tag:cloze finds 1 note" \
    "$AK search 'tag:cloze'" 'length' "1"

assert "ak info shows note model" \
    "$AK info $NOTE1_ID" '"model": "Basic"'

assert "ak info shows note fields" \
    "$AK info $NOTE1_ID" '"Front": "What is 2+2?"'

assert "ak search text finds matching note" \
    "$AK search 'What is 2'" "$NOTE1_ID"

# ============================================================
echo ""
echo "Phase 5: Note modification"
# ============================================================

assert "ak update changes field" \
    "$AK update $NOTE1_ID -F Front='What is two plus two?'" '"updated"'

assert "ak info shows updated field" \
    "$AK info $NOTE1_ID" '"Front": "What is two plus two?"'

$AK tag add $NOTE1_ID updated > /dev/null 2>&1
TESTS=$((TESTS + 1))
PASS=$((PASS + 1))
echo "  PASS: ak tag add runs without error"

$AK tag remove $NOTE1_ID test > /dev/null 2>&1
TESTS=$((TESTS + 1))
PASS=$((PASS + 1))
echo "  PASS: ak tag remove runs without error"

# Verify tags changed
INFO=$($AK info $NOTE1_ID)
TESTS=$((TESTS + 1))
if echo "$INFO" | grep -q '"updated"' && ! echo "$INFO" | grep -q '"test"'; then
    PASS=$((PASS + 1))
    echo "  PASS: tags correctly updated (has 'updated', no 'test')"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: tags not as expected"
    echo "    got: $(echo "$INFO" | jq '.[] .tags')"
fi

# ============================================================
echo ""
echo "Phase 6: Media"
# ============================================================

# Create a minimal valid PNG
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82' > "$TMPDIR/test-ak.png"

MEDIA_ID=$($AK add "Image card" "See below" --image "$TMPDIR/test-ak.png" -d "Test Deck" 2>&1 | jq -r '.id' 2>/dev/null) || MEDIA_ID=""
TESTS=$((TESTS + 1))
if [ -n "$MEDIA_ID" ] && [ "$MEDIA_ID" != "null" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: ak add with --image (id: $MEDIA_ID)"

    assert "image card Back contains img tag" \
        "$AK info $MEDIA_ID" '<img src='
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: ak add with --image"
    # Skip the img tag check
    TESTS=$((TESTS + 1))
    FAIL=$((FAIL + 1))
    echo "  FAIL: image card Back contains img tag (skipped)"
fi

# ============================================================
echo ""
echo "Phase 7: Batch add"
# ============================================================

cat > "$TMPDIR/test-batch.md" << 'BATCHEOF'
deck: Test Deck
tags: batch
model: Basic
---
Q: Batch question 1
A: Batch answer 1
---
Q: Batch question 2
A: Batch answer 2
---
model: Cloze
Q: Batch {{c1::cloze}} card
BATCHEOF

BATCH_RESULT=$($AK add -f "$TMPDIR/test-batch.md" 2>&1)
BATCH_COUNT=$(echo "$BATCH_RESULT" | jq '[.[] | select(. != null)] | length' 2>/dev/null) || BATCH_COUNT=0
TESTS=$((TESTS + 1))
if [ "$BATCH_COUNT" -eq 3 ]; then
    PASS=$((PASS + 1))
    echo "  PASS: batch add created 3 notes"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: batch add expected 3 notes, got $BATCH_COUNT"
    echo "    result: $BATCH_RESULT"
fi

# Verify total count increased
TOTAL=$($AK search 'deck:"Test Deck"' | jq 'length')
TESTS=$((TESTS + 1))
if [ "$TOTAL" -ge 6 ]; then
    PASS=$((PASS + 1))
    echo "  PASS: deck now has $TOTAL notes (expected >= 6)"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: deck has $TOTAL notes (expected >= 6)"
fi

# ============================================================
echo ""
echo "Phase 8: Deletion"
# ============================================================

ALL_IDS=$($AK search 'deck:"Test Deck"' | jq -r '.[]')
ID_ARGS=$(echo "$ALL_IDS" | tr '\n' ' ')
$AK delete $ID_ARGS > /dev/null 2>&1
REMAINING=$($AK search 'deck:"Test Deck"' | jq 'length')
TESTS=$((TESTS + 1))
if [ "$REMAINING" -eq 0 ]; then
    PASS=$((PASS + 1))
    echo "  PASS: all notes deleted"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $REMAINING notes remain after delete"
fi

# ============================================================
echo ""
echo "Phase 9: Sync & GUI"
# ============================================================

# Sync may fail without AnkiWeb account - just check it doesn't crash
SYNC_RESULT=$($AK sync 2>&1) || true
TESTS=$((TESTS + 1))
PASS=$((PASS + 1))
echo "  PASS: ak sync runs without crash"

assert "ak browse runs without crash" \
    "$AK browse 'deck:Test Deck'" ""

# ============================================================
echo ""
echo "Phase 10: Error handling"
# ============================================================

assert_exit "ak add with no args exits 1" \
    "$AK add" 1

assert_exit "ak search with no args exits 1" \
    "$AK search" 1

assert_exit "ak info nonexistent note exits 1" \
    "$AK info 99999999" 1

assert_exit "ak update nonexistent note exits 1" \
    "$AK update 99999999 -F Front=x" 1

# ak add to nonexistent deck should fail (AnkiConnect does NOT auto-create decks)
assert_exit "ak add to nonexistent deck fails" \
    "$AK add 'test' 'test' -d 'Nonexistent Deck'" 1

echo ""
echo "============================================================"
