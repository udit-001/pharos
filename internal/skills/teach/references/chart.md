# Charts — Vega-Lite Authoring Recipe

Use for: **quantitative data on axes** — comparisons, distributions, trends, correlations.

Don't use for: relational structure (mermaid diagram) or equations (katex). To plot a function curve like y = sin(x), sample points into a line mark; show the equation itself in katex.

## Chart-type selection

Pick the mark from the shape of the data, not from what "looks nice":

| Data shape | Mark | Example |
|---|---|---|
| Category → value | `bar` | Quiz accuracy by topic |
| Ordered/continuous x → value | `line` | Accuracy across attempts |
| Two continuous variables | `point` | Study time vs score |
| Distribution of one variable | `bar` + `bin` encoding | Score histogram |
| Multiple series over a shared axis | `line` + `color` encoding | Trends by topic, side by side |
| Part of a whole | `bar` (stacked), **not** `arc` | Time spent per activity |

> Prefer bars over pie/donut (`arc`) — the human eye compares lengths far better than angles. Use `arc` only when the parts number ≤4 and every slice is >10%.

## Authoring rules

| Rule | Why |
|---|---|
| Always set `"width": "container"` | Chart fills its card; responsive |
| Never set colour values in the spec | Nord palette comes from `vega-theme.js` config — overriding it breaks dark mode |
| Keep ≤7 categorical series | The Nord categorical palette has 7 colours; beyond that, colours repeat and confuse |
| Embed data inline (`data.values`) | Lessons must work offline — no external URLs |
| Add `"tooltip": true` to the mark | Hover reveals exact values without clutter |
| >25 bars on a category axis | Switch to a line/area or group into bins — crowded bars are illegible |
| Each `<div class="chart">` gets its own `<script type="application/json">` | One spec per chart, linked by `data-vega` → script `id` |

## Skeleton — bar chart

```html
<div class="chart" data-vega="accuracy-chart"></div>
<script type="application/json" id="accuracy-chart">
{
  "$schema": "https://vega.github.io/schema/vega-lite/v6.json",
  "width": "container", "height": 220,
  "mark": { "type": "bar", "cornerRadiusEnd": 4, "tooltip": true },
  "encoding": {
    "x": { "field": "topic", "type": "nominal", "sort": "-y", "title": null, "axis": { "labelAngle": -30 } },
    "y": { "field": "score", "type": "quantitative", "title": "Score (%)", "scale": { "domain": [0, 100] } }
  },
  "data": { "values": [
    { "topic": "Algebra", "score": 86 },
    { "topic": "Calculus", "score": 64 },
    { "topic": "Geometry", "score": 92 }
  ] }
}
</script>
```

## Worked specs

### Line — trend over a continuous axis

```json
{
  "width": "container", "height": 220,
  "mark": { "type": "line", "strokeWidth": 2.5, "tooltip": true },
  "encoding": {
    "x": { "field": "attempt", "type": "quantitative", "title": "Attempt", "scale": { "domain": [1, 5] } },
    "y": { "field": "score", "type": "quantitative", "title": "Score", "scale": { "domain": [0, 100] } }
  },
  "data": { "values": [
    { "attempt": 1, "score": 52 },
    { "attempt": 2, "score": 61 },
    { "attempt": 3, "score": 73 },
    { "attempt": 4, "score": 79 },
    { "attempt": 5, "score": 88 }
  ] }
}
```

### Point — correlation between two variables

```json
{
  "width": "container", "height": 220,
  "mark": { "type": "point", "filled": true, "size": 60, "opacity": 0.8, "tooltip": true },
  "encoding": {
    "x": { "field": "study", "type": "quantitative", "title": "Study time (min)", "scale": { "domain": [0, 110] } },
    "y": { "field": "score", "type": "quantitative", "title": "Score", "scale": { "domain": [0, 105] } }
  },
  "data": { "values": [
    { "study": 20, "score": 45 },
    { "study": 45, "score": 68 },
    { "study": 80, "score": 82 }
  ] }
}
```

### Histogram — distribution via binning

```json
{
  "width": "container", "height": 220,
  "mark": { "type": "bar", "cornerRadiusEnd": 3, "tooltip": true },
  "encoding": {
    "x": { "bin": { "maxbins": 20 }, "field": "score", "type": "quantitative", "title": "Score" },
    "y": { "aggregate": "count", "type": "quantitative", "title": "Students" }
  },
  "data": { "values": [
    { "score": 55 }, { "score": 62 }, { "score": 71 }, { "score": 78 }, { "score": 85 }
  ] }
}
```

### Grouped bar — two series side by side

Use `xOffset` (not `column` faceting — faceting ignores `width: "container"` and overflows the card):

```json
{
  "width": "container", "height": 220,
  "mark": { "type": "bar", "cornerRadiusEnd": 3, "tooltip": true },
  "encoding": {
    "x": { "field": "topic", "type": "ordinal", "title": null },
    "xOffset": { "field": "term", "type": "nominal" },
    "y": { "field": "score", "type": "quantitative", "title": "Score", "scale": { "domain": [0, 100] } },
    "color": { "field": "term", "type": "nominal", "legend": { "orient": "top", "title": null } }
  },
  "data": { "values": [
    { "topic": "Algebra", "term": "Term 1", "score": 68 },
    { "topic": "Algebra", "term": "Term 2", "score": 82 },
    { "topic": "Calculus", "term": "Term 1", "score": 54 },
    { "topic": "Calculus", "term": "Term 2", "score": 71 }
  ] }
}
```
