# Pull Request

## Resumo

<!-- 1-2 frases descrevendo o objetivo da mudança. -->

## Mudanças

<!-- Bullet list das alterações principais. -->
-
-

## Testes

**How tested:**

<!-- Descreva como você validou a mudança (manual, automatizado, cenário). -->

**`go test ./...`:**

<!-- Cole o resultado resumido (ok / FAIL / skipped). -->

```
```

## Docs

<!-- Marque o que se aplica. -->
- [ ] Atualizei `openapi.yaml`
- [ ] Atualizei `ARCHITECTURE.md`
- [ ] Atualizei `README.md`
- [ ] Não havia mudança de documentação aplicável

## Checklist

- [ ] `gofmt` e `go vet` passando
- [ ] Testes locais verdes (`go test ./...`)
- [ ] CI verde (job `backend build + tests`)
- [ ] Migrations idempotentes (se aplicável)
- [ ] `sqlc generate` rodado e arquivos gerados commitados (se aplicável)
- [ ] `openapi.yaml` atualizado (se a API mudou)
