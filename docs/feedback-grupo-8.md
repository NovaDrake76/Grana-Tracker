# Resposta ao feedback do Grupo 8

> Análise das sugestões em `docs/Sugestões para requisitos do grupo 1.pdf`.
> Time DIM0518 — Breno Jalmir, Felippe Zanathan, Nathan Araújo.

O Grupo 8 aplicou a técnica dos **Seis Chapéus do Pensamento** (azul / amarelo / preto)
sobre dois requisitos funcionais nossos: **RF11 — Visualização das Carteiras** e
**RF24 — Distribuição por Tipo de Ativo**. Este documento percorre, ponto a ponto,
cada sugestão e indica o destino: já entregue, implementado nesta rodada,
aceito como roadmap futuro ou recusado com justificativa.

## Resumo executivo

| Categoria | Quantidade |
|---|---|
| Sugestões implementadas nesta rodada | 3 |
| Sugestões já em produção (endossadas) | 6 |
| Sugestões aceitas como roadmap futuro | 4 |
| Sugestões deferidas/rejeitadas (com justificativa) | 4 |
| **Total de pontos analisados** | **17** |

Legenda de status usada abaixo:

- **ENTREGUE** — já estava em produção antes da revisão; o feedback endossa.
- **IMPLEMENTADO NESTA RODADA** — código novo motivado direta ou indiretamente pelo grupo 8.
- **ROADMAP** — sugestão pertinente, fora do escopo desta entrega final, registrada para próxima iteração.
- **REJEITADO** — analisada e descartada (com motivo técnico ou de produto).

---

## RF11 — Visualização das Carteiras

Requisito de prioridade **Alta**. Permite ao usuário ver, em um lugar só, todas as
carteiras (reais e simuladas) que ele criou.

### Chapéu azul — Júlia Lima

**1. Tela única com filtros entre carteiras reais e simuladas (tabs).** — **ENTREGUE**
Arquivo: `frontend/src/app/dashboard/portfolios/page.tsx`.
A rota `/dashboard/portfolios` já é uma página única. No topo há três abas — **Todos**,
**Reais**, **Simulados** — com contagem ao lado de cada uma. A filtragem é
client-side sobre a lista já carregada, então é instantânea.

**2. Informações mínimas em cada card: nome, tipo, valor investido e rentabilidade.** — **IMPLEMENTADO NESTA RODADA**
Antes da revisão, o card mostrava nome, badge de tipo, valor investido, barra de
alocação mini, contagem de posições e datas — faltava justamente o item
**rentabilidade** que o chapéu azul pediu. Nesta rodada o card passou a calcular,
via hook `frontend/src/hooks/usePriceMap.ts`, o **valor atual de mercado** de cada
posição (`qty × preço atual`) e exibir o **ganho/perda absoluto e percentual** com
cor (verde para ganho, vermelho para perda). Se a cotação de um ativo não está
disponível, faz fallback para o valor investido — o card nunca quebra.

**3. Reservar etapas para protótipos e testes.** — **ENTREGUE (metodológico)**
Endossamos. O fluxo de RF11 foi prototipado de cabeça em Sprint 1, refinado
visualmente nos Sprints 2 e 3 e validado por testes manuais a cada fim de sprint.
Não há ainda framework automatizado no frontend — está no roadmap (ver final).

### Chapéu amarelo — Kézia Lima

**4. Tela unificada simplifica o uso para iniciantes.** — **ENTREGUE (endossado)**
Esse é, literalmente, o nosso objetivo de UX no dashboard: um iniciante não
deveria precisar saber se está olhando renda real ou jogo de simulação para
entender quanto tem. A aba **Todos** é o default exatamente por isso.

**5. Futuro: gráficos de evolução por carteira já na listagem.** — **REJEITADO (com nuance)**
Já temos gráfico de evolução, mas **na página de detalhe** da carteira
(`frontend/src/components/PerformanceChart.tsx`, usado pela US06). Embutir um
mini-chart Recharts em **cada card** da listagem traria dois problemas concretos:
(a) poluição visual quando o usuário tem 5+ carteiras, competindo com o donut
geral logo abaixo; (b) custo de render — Recharts não é barato e cada card
abriria seu próprio container responsivo. A alternativa viável seria um sparkline
SVG puro (sem lib), mas isso fica como roadmap explícito abaixo.

### Chapéu preto — Rayssa Cavalcante

**6. Com muitas carteiras a tela pode ficar cramped; considerar paginação.** — **ROADMAP**
Concordamos com o risco em abstrato, mas no perfil de uso real (usuário típico do
Grana Tracker tem 1 a 5 carteiras: uma real principal + simulações para testar
estratégias) a tela hoje cabe sem rolagem horizontal e respira bem no grid
`md:grid-cols-2 lg:grid-cols-3`. Quando o número médio de carteiras por usuário
ultrapassar ~10, entra **infinite scroll** (mais natural que paginação numerada
para listas curtas e visuais). Pré-requisito barato de adicionar pois o endpoint
`GET /portfolios` já está pronto para aceitar query params.

**7. Distinção visual entre carteiras por cores/nomes.** — **ENTREGUE**
Cada card tem uma **accent-bar** colorida no topo, definida pelo tipo:
**azul** para carteiras reais e **violeta** para simuladas. O badge logo abaixo do
nome reforça a distinção textualmente. Isso é diferente de cor por carteira
individual (próximo item), mas resolve o caso mais frequente de confusão (real vs
brincadeira).

**8. Definir quais informações ficam no card (saldo, tipo, crescimento?).** — **IMPLEMENTADO NESTA RODADA**
A pergunta foi respondida diretamente: o card agora contém **nome, tipo (badge +
accent-bar), valor investido, valor atual, rentabilidade colorida, barra de
alocação mini, número de posições e data de criação**. Foi um exercício
deliberado de não inflar mais — descartamos mostrar contagem de transações e
data da última operação para manter densidade visual.

**9. Ordenação configurável pelo usuário (mais recente, saldo, rendimento, risco, tipo).** — **IMPLEMENTADO NESTA RODADA**
Adicionado dropdown de ordenação ao lado das abas, com seis opções:
**Mais recentes, Mais antigos, Maior saldo, Menor saldo, Maior ganho, Nome A-Z**.
Cobre quatro dos cinco eixos sugeridos. Não incluímos **ordenação por risco**
porque ainda não calculamos volatilidade/beta por carteira — entraria no épico
de métricas avançadas. Ordenação por **tipo** já é coberta pelas abas, então
seria redundante no dropdown.

---

## RF24 — Distribuição por Tipo de Ativo

Requisito de prioridade **Alta**. Mostra como o patrimônio do usuário está
fatiado entre as classes de ativo (ações, ETF, cripto, etc).

### Chapéu azul — Júlia Lima

**10. Definir os tipos suportados: ações, cripto, ETFs, índices, fundos imobiliários, renda fixa.** — **PARCIALMENTE ENTREGUE + ROADMAP**
Hoje o sistema suporta **quatro tipos**: `stock`, `crypto`, `etf`, `index`. Está
declarado como `CHECK constraint` no schema do banco, então o backend rejeita
qualquer valor fora desse conjunto. **Não suportamos FII e renda fixa** nesta
entrega — adicioná-los exigiria nova fonte de preço (FIIs precisam de feed B3 ao
vivo via algo como BRAPI; renda fixa precisa de marcação a mercado por papel via
ComDinheiro/ANBIMA). Fica no roadmap como **épico de novas classes de ativo**.

**11. Cálculo com base no valor ATUAL de mercado (não valor investido).** — **IMPLEMENTADO NESTA RODADA**
Esse foi um dos ganhos mais relevantes desta rodada. Antes, o donut e a legenda
de `AllocationDonut` (`frontend/src/components/AllocationDonut.tsx`) eram
calculados sobre o valor investido (`qty × preço médio de compra`). Agora a
fonte de verdade é `qty × preço atual de mercado`, usando o mesmo `usePriceMap`
do RF11. Quando a cotação não está disponível para um ticker, mantém-se o valor
investido como fallback transparente — assim a soma continua representando o
patrimônio real do usuário no momento da visualização.

**12. Apresentar como pizza/barras + tabelas com valores e %.** — **ENTREGUE**
A home (`/dashboard`) já mostra **donut** (gráfico de rosca) + **barras por
portfolio** lado a lado. A legenda do donut, em `AllocationLegend`, mostra para
cada tipo: o **valor absoluto em R$** e o **percentual relativo**. É exatamente
a combinação visual + tabular sugerida.

**13. Testes para garantir que a soma das porcentagens fecha em 100%.** — **ROADMAP**
Sugestão totalmente pertinente — é o tipo de bug silencioso que aparece quando
algum ativo é classificado com tipo `null` e some da soma. Hoje **não existe
framework de teste no frontend**. O backend não cobre esse teste porque a
agregação por tipo é feita 100% client-side (vem de `usePriceMap` + posições).
Roadmap concreto: adicionar **Vitest** ao frontend e começar exatamente por um
teste unitário do reducer de agregação garantindo `sum(percentages) === 100`
(com tolerância de arredondamento) para um conjunto fixo de posições.

### Chapéu amarelo — Kézia Lima

**14. Endosso à visão de diversificação visível.** — **ENTREGUE (endossado)**
O donut foi escolhido justamente por comunicar diversificação num relance — é
muito mais intuitivo que uma tabela bruta para responder "estou concentrado em
algum tipo?". O feedback confirma que a leitura está acontecendo como esperado.

**15. Futuro: metas de alocação por tipo, com indicador acima/abaixo da meta.** — **ROADMAP**
Excelente sugestão de produto, fora do escopo desta entrega final. Implica em:
modelar `allocation_targets` por usuário no banco, UI para definir as metas,
overlay no donut mostrando o desvio (por exemplo, anel externo cinza marcando o
alvo). Entra como **épico de planejamento financeiro** na próxima iteração.

### Chapéu preto — Rayssa Cavalcante

**16. Que tipo de gráfico usar: barras, pizza ou linha?** — **ENTREGUE (com decisão documentada)**
Decisão tomada: **donut** para a fatia atual + **barras horizontais** para
comparar carteiras lado a lado. Linha foi descartada porque RF24 é uma foto do
**estado atual**, não de evolução temporal (evolução é o RF de PerformanceChart).
Pizza vs donut: optamos por donut porque o centro vazio acomoda o **total
agregado em R$**, que era informação faltante na primeira iteração.

**17. Métrica em % ou em valor? Todos OU dividido por carteira?** — **ENTREGUE**
Resposta: **ambas as métricas, em duas visualizações complementares**. O donut
de cima é **agregado de todas as carteiras** com legenda mostrando valor **e**
percentual. As barras horizontais embaixo são **por carteira individual**, para
que o usuário compare se a carteira simulada está mais arriscada que a real, por
exemplo. Resolve diretamente as duas perguntas do chapéu preto em layout único.

---

## Roadmap evoluído a partir deste feedback

Listado em ordem decrescente de prioridade para a próxima iteração:

1. **Vitest no frontend + teste de invariância da agregação (`sum(%) === 100`).**
   Item 13. Baixo custo, alto valor — destrava cobertura para todo o módulo de
   visualização e gráficos.
2. **Sparkline SVG nos cards de carteira.** Item 5, repensado sem Recharts.
   Cabe em ~60 linhas de código sem dependência nova, sem o custo de render do
   Recharts. Atende ao chapéu amarelo sem cair na rejeição feita acima.
3. **Suporte a FII e renda fixa.** Item 10. Bloqueado por fonte de preço;
   começar avaliando BRAPI (gratuito até certo volume) para FII e marcação a
   mercado simplificada para renda fixa pré-fixada.
4. **Metas de alocação por tipo com indicador above/below.** Item 15. Épico de
   planejamento — depende dos itens 2 e 3 idealmente, para a meta cobrir todas
   as classes de ativo.
5. **Infinite scroll na listagem de carteiras.** Item 6. Gatilho: usuário médio
   passar de ~10 carteiras. Métrica acionável, não premature optimization.

---

## Agradecimentos

Obrigado ao Grupo 8 — Júlia Lima, Kézia Lima e Rayssa Cavalcante — pela
aplicação cuidadosa dos Seis Chapéus em **dois** dos nossos requisitos de maior
visibilidade no produto. O chapéu preto, em particular, gerou três das três
melhorias entregues nesta rodada (rentabilidade no card, ordenação configurável,
cálculo do donut por preço de mercado), todas embarcadas na versão final do
Grana Tracker. Sugestões deferidas viraram itens nomeados de roadmap em vez de
ficarem como "vamos ver depois" — esse é o melhor uso que conseguimos dar a uma
revisão cruzada bem feita.
