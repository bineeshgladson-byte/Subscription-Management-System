# ADR-001: Quotations are stored in the `subscriptions` table

**Status:** Proposed (awaiting team approval)

**Context.** The brief gives quotations and subscriptions the same lifecycle
(Draft → Quotation → Confirmed → Active → Closed).

**Decision.** One `subscriptions` row moves through every status. `quotation_number`
is assigned at creation, `subscription_number` at confirmation. Quotation screens
and `/quotations` endpoints are a filtered view (`status IN ('draft','quotation')`).

**Consequences.** No copying of lines on confirmation, one set of total calculations,
one status machine. If rejected, add `quotations` + `quotation_items` tables and a
copy step in `confirm`.
