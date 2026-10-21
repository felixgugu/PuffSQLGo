# PuffSQL Frontend

Vue 3 + TypeScript frontend for the Go/Wails PuffSQL desktop client.

The UI uses Pinia for state, Monaco for T-SQL editing, Tabulator 6 for virtualized result grids,
AntV X6 for ER diagrams and Tailwind CSS for the visual system. Desktop calls are isolated in
`src/services/api.ts` and use the generated bindings in `wailsjs/go/main/App`.

Browser development uses the mock backend in `src/services/api.ts`:

```powershell
npm install
npm run dev
```

Type-check, test and build:

```powershell
npm run typecheck
npm test
npm run build
```

Use `wails dev` from the repository root when the Go backend is required.
