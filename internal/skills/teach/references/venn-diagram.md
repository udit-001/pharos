# Venn Diagrams — Callout Pattern

Use for: comparing two sets/concepts, showing overlap, "where A meets B" in a lesson.

## Callout rules

When text doesn't fit a region, it goes in a **callout box** connected by a dashed leader line. Never shrink font or cram text across a circle stroke. Apply every row before shipping.

| Condition | Action |
|---|---|
| Text ≤8 words inside a circle or overlap | Place inside the region |
| Text >8 words in any region | External callout box beside the diagram |
| Overlap with ≥2 bullet points | Dedicated callout box below the diagram |
| Text would cross a circle stroke | Move to a callout — illegible across a stroke |
| Font would need to be <11px to fit | Move to a callout — never shrink below 11px |
| Diagram has 4+ overlapping sets | Use a matrix or table instead |
| No dark mode variant | Add `[data-theme="dark"]` CSS overrides matching light theme |

## SVG skeleton — 2 circles with callouts

```svg
<svg viewBox="0 0 900 420" xmlns="http://www.w3.org/2000/svg">
  <style>
    .venn-circle-left  { fill: rgba(52,119,235,0.05); stroke: #3477eb; stroke-width: 1.5; }
    .venn-circle-right { fill: rgba(235,104,52,0.05); stroke: #eb6834; stroke-width: 1.5; }
    .venn-clip-fill-l  { fill: rgba(52,119,235,0.07); }
    .venn-clip-fill-r  { fill: rgba(235,104,52,0.07); }
    .venn-circle-label { font-size: 15px; font-weight: 700; fill: #1e1e2e; }
    .venn-overlap-label { font-size: 12px; font-weight: 600; fill: #555; }
    .callout-title     { font-size: 13px; font-weight: 700; fill: #1e1e2e; }
    .callout-bullet    { font-size: 12px; fill: #444; }
    .callout-box { fill: none; stroke: rgba(0,0,0,0.15); stroke-width: 1; rx: 8; }
    .leader { stroke: rgba(0,0,0,0.2); stroke-width: 1; stroke-dasharray: 4 3; }

    [data-theme="dark"] .venn-circle-left  { fill: rgba(52,119,235,0.08); stroke: #5a8ff0; }
    [data-theme="dark"] .venn-circle-right { fill: rgba(235,104,52,0.08); stroke: #ff8860; }
    [data-theme="dark"] .venn-clip-fill-l  { fill: rgba(52,119,235,0.04); }
    [data-theme="dark"] .venn-clip-fill-r  { fill: rgba(235,104,52,0.04); }
    [data-theme="dark"] .venn-circle-label { fill: #f0f0f0; }
    [data-theme="dark"] .venn-overlap-label { fill: #d0d0d0; }
    [data-theme="dark"] .callout-title    { fill: #f0f0f0; }
    [data-theme="dark"] .callout-bullet   { fill: #ccc; }
    [data-theme="dark"] .callout-box { stroke: rgba(255,255,255,0.15); }
    [data-theme="dark"] .leader { stroke: rgba(255,255,255,0.2); }
  </style>

  <defs>
    <clipPath id="clip-l"><circle cx="380" cy="140" r="110"/></clipPath>
    <clipPath id="clip-r"><circle cx="520" cy="140" r="110"/></clipPath>
  </defs>

  <!-- Circles -->
  <circle cx="380" cy="140" r="110" class="venn-circle-left"/>
  <circle cx="520" cy="140" r="110" class="venn-circle-right"/>

  <!-- Focal highlight on intersection -->
  <g clip-path="url(#clip-l)">
    <g clip-path="url(#clip-r)">
      <rect width="900" height="420" class="venn-clip-fill-l"/>
      <rect width="900" height="420" class="venn-clip-fill-r"/>
    </g>
  </g>

  <!-- Short labels inside circles -->
  <text class="venn-circle-label" x="325" y="140" text-anchor="middle" dominant-baseline="middle">Set A</text>
  <text class="venn-circle-label" x="575" y="140" text-anchor="middle" dominant-baseline="middle">Set B</text>

  <!-- Short overlap label -->
  <text class="venn-overlap-label" x="450" y="140" text-anchor="middle" dominant-baseline="middle">Shared</text>

  <!-- Left callout box -->
  <rect x="10" y="55" width="200" height="170" class="callout-box"/>
  <text class="callout-title" x="110" y="80" text-anchor="middle">Set A View</text>
  <text class="callout-bullet" x="28" y="105">•  Bullet one</text>
  <text class="callout-bullet" x="28" y="127">•  Bullet two</text>

  <!-- Right callout box -->
  <rect x="690" y="55" width="200" height="170" class="callout-box"/>
  <text class="callout-title" x="790" y="80" text-anchor="middle">Set B View</text>
  <text class="callout-bullet" x="708" y="105">•  Bullet one</text>
  <text class="callout-bullet" x="708" y="127">•  Bullet two</text>

  <!-- Leader lines -->
  <line x1="210" y1="140" x2="270" y2="140" class="leader"/>
  <line x1="630" y1="140" x2="690" y2="140" class="leader"/>

  <!-- Overlap callout below -->
  <line x1="450" y1="250" x2="450" y2="290" class="leader"/>
  <rect x="310" y="290" width="280" height="120" class="callout-box"/>
  <text class="callout-title" x="450" y="315" text-anchor="middle">Shared</text>
  <text class="callout-bullet" x="328" y="340">•  Shared point</text>
</svg>
```

## Layout

| Element | Position |
|---|---|
| Left circle | cx=380, cy=140, r=110 |
| Right circle | cx=520, cy=140, r=110 |
| Left callout | x=10, y=55, w=200, h=170 |
| Right callout | x=690, y=55, w=200, h=170 |
| Overlap callout | x=310, y=290, w=280, h=120 |
