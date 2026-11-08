# TODO – needforheat-manual-server

Open work for this repository, distilled from the old NeedForHeat Trello board, including the discussions on the cards (paraphrased and regrouped) and from the 2026 maintenance review.

Items that touch several repositories are written out in full in one *lead* repository; the others list them with a link (→). Keep the details up to date in the lead repository only.

## Soon: robustness and correctness

- [ ] <a id="msrobust"></a>**Robust manual server.** A wrong folder layout in one repository should not break the rest; use a default `template.html` when a repository has none (and document it); return 404 instead of a redirect loop for missing pages. Also affects: needforheat-manuals. *(Trello #842, #843, #834)*

## Later: features and ideas

- [ ] → Finish the rename from Twomes to NeedForHeat — see [needforheat-generic-firmware](https://github.com/energiegedrag-nl/needforheat-generic-firmware/blob/main/TODO.md#rename) *(Trello #1030)*
- [ ] → CONTRIBUTING.md with the branching workflow — see [needforheat-generic-firmware](https://github.com/energiegedrag-nl/needforheat-generic-firmware/blob/main/TODO.md#contrib) *(Trello #394, #1032)*
- [ ] → Remove the duplicated header in manuals, FAQ and privacy pages — see [needforheat-manuals](https://github.com/energiegedrag-nl/needforheat-manuals/blob/prd/TODO.md#header) *(Trello #789)*
- [ ] → Language in the path, so relative links work — see [needforheat-manuals](https://github.com/energiegedrag-nl/needforheat-manuals/blob/prd/TODO.md#lang) *(Trello #1002)*
