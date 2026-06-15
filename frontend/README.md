# Grana Tracker — Frontend

Aplicação web em Next.js 16 (App Router) com Chakra UI v3, que consome a API REST do backend Go via JWT Bearer.

## Setup

- Node 20+
- `npm install`

## Variáveis de ambiente

Crie um `.env.local` (ou copie `.env.example`):

- `NEXT_PUBLIC_API_URL` — URL base da API do backend, incluindo o prefixo `/api` (ex.: `http://localhost:8080/api`). Se ausente, o cliente usa `http://localhost:8080/api` como fallback.

## Scripts

- `npm run dev` — sobe o servidor de desenvolvimento em `http://localhost:3000`
- `npm run build` — build de produção
- `npm run start` — serve o build de produção
- `npm run lint` — roda o ESLint

## Estrutura

```
src/
  app/
    layout.tsx            # root layout + providers
    providers.tsx         # Chakra + AuthContext
    page.tsx              # landing
    login/                # tela de login
    register/             # tela de cadastro
    dashboard/
      layout.tsx
      page.tsx            # visão geral
      portfolios/         # listagem, /new e /[id]
  components/             # StatCard, Icons, etc.
  context/
    AuthContext.tsx       # estado de autenticação
  lib/
    api.ts                # fetch wrapper com Bearer + refresh
    theme.ts              # tema Chakra
    toaster.ts            # toasts
  types/                  # tipos compartilhados
```

## Autenticação

- Os tokens são armazenados no `localStorage` do browser como `access_token` e `refresh_token` (ver `src/lib/api.ts`).
- Toda requisição autenticada envia o header `Authorization: Bearer <access_token>`.
- Em resposta `401`, o wrapper tenta uma vez renovar o par de tokens via `POST /auth/refresh` (rotação de refresh token) e reexecuta a chamada original; se a renovação falhar, limpa o storage e redireciona para `/login`.

## Links

- Visão geral do projeto: `../README.md` na raiz do repositório.
- Spec da API: `../docs/openapi.yaml`.
