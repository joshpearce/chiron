# design-sync notes

- This repo is NOT a React design system (Go server + QML e-ink client);
  there are no components to convert or bundle. The Claude Design project
  "Chiron" is a CONTEXT project: it carries DESIGN-brief.md as guidelines
  plus the before-captures from rmpp/design-captures/, so the design agent
  designs against the brief with its generic components. The specs it
  produces are implemented by hand in the server page CSS + QML and
  verified with the drive harness (see DESIGN-ui.md).
- A standard component sync will never apply here unless the client is
  someday rebuilt on a React design system.
