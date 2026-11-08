# Mobile layout repair

The room footer must keep navigation, board-view controls and turn actions
inside the viewport without overlapping. Groups may wrap onto another row;
the clue text, count and submit action should share a row on phones.

Moderator settings must fit their container, including long profile names and
wordpack labels. A fieldset's default minimum content width must not stretch
the settings grid. Controls should shrink and labels wrap inside the panel.

The footer uses a three-column grid for clue text, count and submit. Navigation
and board-view groups retain their width and wrap as groups when space runs out.
The collapse button sits above the content so the reserved space remains clickable.
Moderator settings explicitly reset the fieldset minimum width, constrain the
profile dropdown, and allow the wordpack grid item to shrink.

## Regression checks

Install browser binaries with `pnpm --dir web exec playwright install chromium firefox`,
then run `pnpm --dir web test:responsive`. The suite starts its own Vite server
on port 5174 and mounts the production components with deterministic fixture data.
It checks clipping (including inner clipping containers), overlapping controls,
button hit targets, clue submission, pass, board view, collapse/expand and
wordpack add/remove. Widths cover 320, 360, 393, 430, 768, 1024 and 1440 CSS pixels;
the 768px case uses a short landscape viewport. All three game modes are covered.

Fixture types can be checked with
`pnpm --dir web exec svelte-check --tsconfig ./e2e/tsconfig.json`.
The `.pw.ts` suffix keeps browser cases out of the Vitest unit suite.

WebKit is configured for environments where its binary works:
`pnpm --dir web exec playwright install webkit` and
`pnpm --dir web exec playwright test --project=webkit`.
The available macOS WebKit binary crashed before page load during this repair;
Safari and physical-device testing remain unverified.

During validation, the browser tests reproduced both original failures before
the CSS changes and passed afterward. The frontend unit suite, component and
fixture type checks, production build, and hosting-script tests also passed.
Run the hosting-script tests with `zsh -f ./self_host_test.zsh` if local shell
initialization interferes with its strict unset-variable handling.

The backend suite exposed an unrelated existing failure in
`TestUsersRoomsMatchesSnapshotsEventsAndChatPersist`: its chat-order assertion
can fail when both messages receive the same millisecond timestamp. No storage
code is changed by this repair.
