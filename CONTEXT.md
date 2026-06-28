# Wordle Solver — kontext projektu

Řešič a analyzátor české varianty hry Wordle (slova s diakritikou). Modul
`github.com/pracj3am/wordle-solver`, Go 1.24.

Tento dokument shrnuje doménové pojmy, architekturu a netriviální invarianty.
Vznikl konsolidací komentářů z kódu — slouží jako jediné místo, kde se tahle
„proč" znalost drží (kód samotný zůstává s minimem komentářů).

## Co to dělá

- **`main.go`** — interaktivní CLI řešič. Po každém tahu zadáš slovo a barvy
  zpětné vazby, nástroj vypíše zbývající slova a pro každý tip metriky
  (obtížnost, „IQ", štěstí).
- **`analyzer/`** — tatáž logika jako knihovna (API vhodné i pro WASM): pro
  odehranou hru spočítá pro každý tip metriky. Hlavní, udržovaná varianta.
- **`first/`** — předpočítá `luck.gob` (statistiky pro 1. tah, viz níže).
- **`hacky/`** — scraper slovníku z `ssjc.ujc.cas.cz` (Příruční slovník
  jazyka českého) — generuje kandidátní slova přes regex s diakritikou.
- **`presmycky/`** — pomůcka na přesmyčky (build-ignored, `//go:build ignore`):
  pro každé použité slovo hledá ve slovníku slova ze stejných písmen.

## Pravidla hry a barvy

Pětipísmenná slova. Zpětná vazba se v CLI zadává znaky (po jednom na pozici):

| znak  | barva    | význam |
|-------|----------|--------|
| `+`   | zelená   | písmeno je na správné pozici (přesný tvar včetně diakritiky) |
| `*`   | modrá    | jako zelená (pozice sedí) **a** písmeno se ve slově vyskytuje ještě aspoň jednou (floor +1) |
| `.`   | oranžová | správné písmeno, špatná pozice (vyskytuje se jinde, aspoň 1×) |
| ` `   | šedá     | písmeno se (v daném počtu) nevyskytuje (přesný počet, typicky 0) |

Frekvenční omezení (`valid` v `progress`) pracují per-písmeno na **základním**
(bezdiakritickém) tvaru: `floor` = „aspoň N výskytů", `exact` = „přesně N
výskytů". Zelená navíc zamyká pozici na **přesný** akcentovaný tvar.

## Diakritika (jádro doménového modelu)

Klíčový rozdíl od anglického Wordle: pracuje se s češtinou a diakritika se
v některých omezeních „skládá" na základní písmeno, v jiných ne.

- Abeceda má **41 znaků** = 26 základních + 15 akcentovaných variant
  (`á č ď é ě í ň ó ř š ť ú ů ý ž`). Viz `dict.Letters`/`dict.Písmena`
  (indexované 0–40) a `dict.Indexes` (rune → index).
- `dict.Conv` (rune → základní rune) a `dict.ConvIndex` (index → základní
  index 0–25) složí akcent na základní písmeno (`á→a`, `ř→r`, `ě→e` …).
  `dict.StripDiacritic` odstraní diakritiku z celého slova.
- **Konvence pojmenování v kódu:** proměnná `písmeno` = znak *s* diakritikou;
  `letter` (a `písm` / `WithoutDiacritics`) = *bez* diakritiky.
- Zelená pozice vyžaduje přesný akcentovaný tvar; oranžová/šedá a všechna
  frekvenční omezení běží nad základním písmenem.

Slovník se ukládá jako **trie** s 41 větvemi na úroveň (`dict.nextLetter`),
slova jsou uložená s diakritikou. `Progress.WordsLeft` prochází trie pětkrát
zanořeně a filtruje přes `valid`.

## Datové soubory

| soubor         | obsah |
|----------------|-------|
| `db.txt`       | slovník bez diakritiky (~2862 slov) |
| `db-hacky.txt` | slovník s diakritikou (~3003 slov) — používá CLI i analyzer |
| `used.txt`     | již použitá denní slova („historie", s diakritikou) |
| `luck.gob`     | předpočítané statistiky pro 1. tah (gob: `luck`, `skillRobot`, `skillHuman`) |

## Klíčové pojmy: `Used`, odpovědi vs. platná slova

- `dict.DictionaryWord.Used` = slovo **NENÍ možná odpověď** (už bylo použité
  jako denní slovo, resp. není v množině odpovědí). Plní se z `history`
  (`used.txt`) při načítání slovníku.
- **Možná odpověď** (answer) = `Used == false`. **Ostatní platné slovo** =
  `Used == true` (smí se hádat, ale nemůže být řešení).
- „Robot" zná i použitá slova, „Human" ne — proto pole `Skills.Human`
  („human nezná použitý slova"). Metriky se počítají v obou variantách.

V `analyzer` se množina odpovědí předává explicitně (`answers []string`) a
`answersToHistory` z ní odvodí `Used` pro všechna slova slovníku.

## Metriky (jeden řádek `analyzer.Row` na tip)

- **Left** — počet všech platných zbývajících slov.
- **LeftAnswers** (v CLI `LeftNotUsed`) — z toho možné odpovědi (`Used=false`).
- **Difficulty** (obtížnost) — jak moc na volbě tipu záleží (rozptyl kvality
  mezi kandidáty). `0` = vynucený tah (zbývá 1 slovo). `-1` = „–" (nedostupné).
- **IQ** (`odds.Skill.Relative`, 0..100) — jak dobrý byl tip mezi možnými;
  100 = nejlepší, 0 = nejhorší. `-1`/„–" když `Difficulty==0` (nehodnotí se).
- **Luck** (štěstí, %) — procento řešení, u nichž by tip dopadl hůř (vyšší =
  větší štěstí). `-1`/„–" když není dost dat.

Výpočet kvality tipu (`odds`): pro slovo se přes všechny možné odpovědi
nasimuluje tah a změří **průměrný počet zbylých slov** (váha; menší = lepší).
`odds.CalculateSkill` z vah seřazených vzestupně spočítá `Difficulty`
(rozptyl vůči nejlepšímu) a `Relative` (relativní skóre 0..100).

`Luck` se počítá z histogramu (`LuckStat`): rozdělení počtu zbylých odpovědí
pro daný tip přes všechna možná řešení — `luckPct` z něj odvodí percentil.

## `luck.gob` a první tah

Metriky se počítají **živě** nad aktuálně zbývajícími slovy. Pro **1. tah** by
ale výpočet nad celým fondem trval moc dlouho, proto se předpočítá do
`luck.gob` (`first/` resp. `analyzer.GenerateLuck`, paralelně přes `NumCPU`).

- `analyzer.NewEngine(dictPath, luckPath, answers)`: když `luckPath != ""` a
  soubor existuje, načte `luck.gob` → 1. tah má luck i difficulty/IQ.
- Bez gobu má 1. tah obtížnost/IQ „–"; analyzer si dopočítá aspoň štěstí
  nad plným fondem (fallback v `Analyze`).

Smyčka `Analyze` věrně kopíruje referenční CLI: metriky pro tip se spočítají
na **konci předchozího kola** nad tehdy zbývajícími slovy (a uloží do
`luckMap`/`skillMap` pro příští iteraci).

## Výpočetní náročnost: `OddsThreshold`

Živý výpočet metrik je ~O(N³) (`calcOdds` pro každé zbylé slovo prochází celý
fond). Proto se nad `Engine.OddsThreshold` kandidátů (default
`defaultOddsThreshold = 150`; v CLI natvrdo `< 1000`) další tah nepočítá živě a
vyjde jako „–". Pro velké fondy to na pomalém CPU trvá desítky sekund; 1. tah
to neřeší, protože má hodnoty z `luck.gob`. `OddsThreshold` jde nastavit zvenčí.

## WASM

`analyzer` a `dict` mají varianty bez souborového systému (`*FromReader` /
`*FromBytes`): slovník i `luck.gob` se předají jako `[]byte`.
- `NewEngineFromBytes(dictData, luckData, answers)` — `luckData` smí být
  prázdné (pak 1. tah „–").
- `LoadDictionaryFromReader` čte vstup **dvakrát** (proto `bytes.NewReader`
  pokaždé znovu).
