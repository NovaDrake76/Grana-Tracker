# Deploy do Grana Tracker (Render + Vercel)

> Guia operacional para subir o produto em produção usando free tiers.
> Última atualização: 2026-06-22.

---

## TL;DR

O backend (Go + Chi) e o Postgres ficam na **Render**, ambos no plano `free`; o frontend (Next.js 16) sobe na **Vercel** no plano Hobby. Tempo total estimado: **15–20 minutos** de cliques se as contas já estiverem criadas. A ordem importa: **Render primeiro**, porque a URL pública do backend precisa ser plugada na variável `NEXT_PUBLIC_API_URL` do Vercel **antes** do build do frontend (variáveis `NEXT_PUBLIC_*` são inlinadas em build time, não em runtime).

---

## Pré-requisitos

- Conta no GitHub com acesso ao repo `NovaDrake76/grana-tracker`.
- Conta na [Render](https://dashboard.render.com) (grátis, autenticação via OAuth GitHub).
- Conta na [Vercel](https://vercel.com) (grátis, autenticação via OAuth GitHub).
- Chave da [Alpha Vantage](https://www.alphavantage.co/support/#api-key) — **opcional**. Sem ela, as cotações de ações/ETFs ficam desligadas (o `AlphaVantageSource` detecta placeholders e curto-circuita), mas o app continua funcional via CoinGecko para cripto.

---

## Limites do free tier (ser honesto)

A banca pode perguntar — vale ter os números na ponta da língua:

- **Render Postgres free:** a instância **expira em 90 dias** a partir da criação. Depois disso é preciso apagar e recriar o banco — os dados de demonstração são perdidos. Anotar a data de provisionamento.
- **Render Web free:** o container **dorme após ~15 min sem tráfego**. A primeira requisição após o sono leva ~30 s para subir o cold start. O frontend pode estourar timeout na primeira chamada se não houver retry.
- **Render free:** logs ficam disponíveis por **24 h**; sem retenção de longo prazo nem backup automático do Postgres.
- **Vercel Hobby:** 100 GB de bandwidth/mês e builds ilimitados. Suficiente com folga para uso acadêmico.

---

## Passo 1 — Backend + Postgres na Render

O repo já tem um `render.yaml` na raiz que declara o **Blueprint**: 1 web service Docker + 1 banco Postgres free. Fluxo click-by-click:

1. Acessar `https://dashboard.render.com` e clicar em **New +** → **Blueprint**.
2. Conectar o GitHub (se ainda não conectado) e selecionar o repositório `grana-tracker`.
3. A Render lê o `render.yaml` e mostra **1 service + 1 database** prontos para serem criados (`grana-tracker-api` + `grana-tracker-db`).
4. Preencher os dois env vars marcados como `sync: false` (placeholders que só o usuário pode setar):
   - `ALPHA_VANTAGE_API_KEY` → colar a chave real, **ou** deixar `your-key` para rodar apenas com CoinGecko.
   - `FRONTEND_URL` → deixar **em branco** por enquanto. Volta-se aqui no Passo 3 com a URL do Vercel.
5. Clicar em **Apply**. A Render aprovisiona o Postgres e dispara o build do Docker — leva **5–8 minutos** na primeira vez.
6. Quando o status virar `Live`, anotar a URL pública do serviço (formato `https://grana-tracker-api-<hash>.onrender.com`).
7. Sanity check com `curl`:

   ```bash
   curl https://<sua-url>.onrender.com/healthz
   # esperado: {"status":"ok"}
   ```

O `JWT_SECRET` é gerado automaticamente pela Render (`generateValue: true`), o `DATABASE_URL` é injetado via `fromDatabase` apontando para a instância recém-criada, e o `PORT` é setado pela própria plataforma (o `main.go` lê `os.Getenv("PORT")`).

---

## Passo 2 — Frontend na Vercel

1. Acessar `https://vercel.com` e clicar em **Add New Project** → **Import Git Repository**.
2. Selecionar `grana-tracker`.
3. **CRÍTICO** — em **Project Settings → General → Root Directory**, setar `frontend`. Sem isso o Vercel tenta buildar a raiz do monorepo (que não tem `next` instalado) e quebra.
4. **Framework Preset:** Next.js (autodetectado quando o Root Directory está correto).
5. Em **Environment Variables**, adicionar:

   ```
   NEXT_PUBLIC_API_URL = https://<sua-url-do-passo-1>.onrender.com/api
   ```

   Atenção ao `/api` no final — o `src/lib/api.ts` espera a base já incluindo o prefixo.
6. Clicar em **Deploy**. Build leva **2–3 minutos**.
7. Anotar a URL pública (formato `https://grana-tracker-<hash>.vercel.app`).

> Observação: como `NEXT_PUBLIC_*` é inlinado em build time, **se a URL do backend mudar depois é preciso redeploy manual** no Vercel para propagar.

---

## Passo 3 — Fechar o circuito (CORS)

Até este ponto o backend ainda não conhece o domínio do frontend, então o CORS bloqueia tudo. Para liberar:

1. Voltar ao dashboard da Render → serviço `grana-tracker-api` → aba **Environment**.
2. Editar `FRONTEND_URL` e colar a URL do Vercel:

   ```
   https://grana-tracker-<hash>.vercel.app
   ```

3. Para manter o dev local funcionando também, usar lista separada por vírgula (o `parseAllowedOrigins` no `router.go` já suporta):

   ```
   https://grana-tracker-<hash>.vercel.app,http://localhost:3000
   ```

4. Salvar. A Render redeploya automaticamente em ~1 min.

---

## Passo 4 — Smoke test end-to-end

Depois que o redeploy do Passo 3 terminar:

1. Abrir a URL do Vercel no navegador.
2. Cadastrar uma conta nova (email válido, senha ≥ 8 caracteres).
3. Logar.
4. Criar um portfólio (`Real` ou `Simulado`).
5. Adicionar um investimento (ex: `BTC`, data recente — o auto-fill de preço deve preencher `purchase_price`).
6. Voltar ao dashboard e conferir que os cards **Patrimônio atual** e **Ganho/Perda** mostram número não-zero.

Se algum passo falhar, ver tabela de troubleshooting abaixo.

---

## Troubleshooting

| Sintoma | Causa provável | Solução |
|---|---|---|
| Frontend mostra erro de CORS no console | `FRONTEND_URL` na Render não bate com o domínio real do Vercel | Corrigir o env var, aguardar o redeploy (~1 min) |
| Primeira requisição leva ~30 s | Cold start do free tier após 15 min ocioso | Comportamento esperado; basta esperar ou usar um ping externo |
| Erro de conexão com o Postgres | Banco free expirou (90 dias) | Apagar a instância, recriar, perde-se os dados de demo |
| Cotações de ações retornam vazio | Quota Alpha Vantage gasta (25 req/dia free) | Esperar reset 00:00 UTC ou trocar de chave |
| Build no Vercel falha com `next: command not found` | Root Directory não está como `frontend` | Ajustar em Project Settings → General |
| `/healthz` retorna `connection refused` | Build ainda rodando, ou Dockerfile não respeita `$PORT` | Esperar build terminar; conferir `main.go` lê `os.Getenv("PORT")` |
| `401` ao tentar logar imediatamente após cadastro | Latência do refresh token entre frontend e backend cold | Recarregar a página uma vez |

---

## Custom domain (opcional)

- **Vercel:** Project → Settings → Domains → **Add Domain** e apontar um CNAME para `cname.vercel-dns.com`.
- **Render:** Service → Settings → **Custom Domain** e apontar um CNAME para o host fornecido.
- Após os domínios ficarem ativos, atualizar `FRONTEND_URL` (Render) e `NEXT_PUBLIC_API_URL` (Vercel) e fazer redeploy do frontend.

---

## Variáveis de ambiente — referência completa

| Variável | Onde mora | Valor exemplo | Observação |
|---|---|---|---|
| `DATABASE_URL` | Render (web service) | injetado via `fromDatabase` | Não setar manualmente; vem do banco do blueprint |
| `JWT_SECRET` | Render (web service) | gerado pela Render | `generateValue: true` no `render.yaml`; ≥32 bytes |
| `PORT` | Render (web service) | injetado pela plataforma | `main.go` lê via `os.Getenv("PORT")` |
| `ALPHA_VANTAGE_API_KEY` | Render (web service) | `XYZ123...` ou `your-key` | Placeholder desativa pricing de ações sem quebrar o app |
| `COINGECKO_BASE_URL` | Render (web service) | `https://api.coingecko.com/api/v3` | Já fixado no `render.yaml` |
| `FRONTEND_URL` | Render (web service) | `https://<vercel>.vercel.app,http://localhost:3000` | Aceita lista separada por vírgula |
| `NEXT_PUBLIC_API_URL` | Vercel (frontend) | `https://<render>.onrender.com/api` | Inlinado em build time; redeploy manual ao mudar |

---

## O que NÃO está no free tier

Vale citar na apresentação para mostrar consciência das limitações:

- **HTTPS em custom domain na Render free:** domínios `.onrender.com` têm HTTPS automático, mas custom domains exigem plano pago para certificado gerenciado.
- **Logs persistentes:** logs ficam apenas 24 h na Render free; sem retenção de longo prazo, sem export estruturado.
- **Backup automático do Postgres:** o plano free não faz snapshot; em caso de perda do banco, recuperar manualmente via `pg_dump` periódico (não automatizado neste projeto).
- **Sleep do web service:** sem opção de manter o container "always-on" no free; para evitar cold start é preciso upgrade para Starter ($7/mês na época da última checagem).
- **Múltiplas regiões / réplicas:** free roda em uma única região (Oregon por padrão); latência fora dos EUA é perceptível.
