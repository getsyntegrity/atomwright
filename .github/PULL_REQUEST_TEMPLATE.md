<!-- ⚠️ READ BEFORE SUBMITTING
  Every PR must be linked to an issue that has the "status:approved" label.
  See CONTRIBUTING.md for the full contribution workflow.
-->

## 🔗 Linked Issue

Closes #

<!-- Replace the # above with the issue number, e.g.: Closes #42 -->

---

## 🏷️ PR Type

- [ ] `type:bug` — Bug fix (non-breaking change that fixes an issue)
- [ ] `type:feature` — New feature (non-breaking change that adds functionality)
- [ ] `type:docs` — Documentation only
- [ ] `type:refactor` — Code refactoring (no functional changes)
- [ ] `type:chore` — Build or tooling changes
- [ ] `type:breaking-change` — Breaking change (changes existing behavior)

---

## 📝 Summary

<!-- What this PR does, and why. -->

---

## 📂 Changes

| File / Area | What Changed |
|-------------|-------------|
| `path/to/file` | Brief description |

---

## 🤖 AI Assistance

Select exactly one. Do not check both.

- [ ] **None** — No material AI assistance was used.
- [ ] **Material assistance used** — Complete the fields below.

**Tool/model (if known):**

**Material scope:**

**Verification performed:**

Trivial formatting, spelling, autocomplete, navigation, and non-substantive
mechanical transformations do not need to be itemized. See
[AI_POLICY.md](../AI_POLICY.md) for the canonical policy.

---

## 🧪 Test Plan

CI runs this exact target on your PR. Run it yourself first and paste the real output:

```bash
make check
```

- [ ] `make check` passes locally
- [ ] Manually verified where automated checks cannot reach

<!-- Paste the actual output above. "go build" alone is not verification —
     it compiles without running tests. `make check` runs gofmt, go vet,
     go mod tidy -diff, go build, go test, and the ADR-0001 architecture
     checks. -->

---

## ✅ Contributor Checklist

- [ ] PR is linked to an issue with `status:approved`
- [ ] PR stays within 400 changed lines, or `size:exception` was applied with a documented rationale
- [ ] Exactly one `type:*` label is applied
- [ ] The commands above were run and their real output is in the Test Plan
- [ ] Nothing contradicts an accepted ADR without a superseding ADR — see [docs/adr](../docs/adr)
- [ ] `internal/domain/*` imports no `internal/application`, `platform/*` or `adapters/*` package
- [ ] `platform/*` imports no `internal/domain/*` package — this rule has no exception
- [ ] `adapters/*` imports `internal/domain/*` only to implement a port interface declared there, or to use a type that port's contract requires — never to call domain logic
- [ ] Documentation updated if necessary, and no document exceeds 300 lines
- [ ] Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
- [ ] Commits do not include `Co-Authored-By` trailers or other AI attribution
- [ ] I understand, reviewed, and take responsibility for the complete submission

---

## 💬 Notes for Reviewers

<!-- Optional: anything reviewers should pay special attention to. -->
