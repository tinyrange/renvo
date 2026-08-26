const numberPattern = String.raw`[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?`;
const labelledValue = new RegExp(String.raw`([A-Za-z_][A-Za-z0-9_.%/()\-]*):\s*(${numberPattern})`, "g");
const plainLine = new RegExp(String.raw`^\s*(${numberPattern})(?:[\s,;]+(${numberPattern}))*\s*$`);

// parsePlotLine accepts the labelled and unlabelled formats understood by the
// Arduino Serial Plotter. Labelled values use name:value and may be separated
// by spaces, tabs, commas, or semicolons. Unlabelled values are named Value 1,
// Value 2, and so on.
export function parsePlotLine(line) {
  const values = [];
  let match;
  let at = 0;
  labelledValue.lastIndex = 0;
  while ((match = labelledValue.exec(line)) !== null) {
    if (!/^[\s,;]*$/.test(line.slice(at, match.index))) return [];
    values.push({ name: match[1], value: Number(match[2]) });
    at = labelledValue.lastIndex;
  }
  if (values.length) {
    if (!/^[\s,;]*$/.test(line.slice(at))) return [];
    return values.every(({ value }) => Number.isFinite(value)) ? values : [];
  }
  if (!plainLine.test(line)) return [];
  const numbers = line.trim().split(/[\s,;]+/).map(Number);
  return numbers.map((value, index) => ({ name: `Value ${index + 1}`, value }));
}

export class SerialPlotter {
  constructor({ capacity = 240, onChange = () => {} } = {}) {
    this.capacity = capacity;
    this.onChange = onChange;
    this.pending = "";
    this.sample = 0;
    this.series = new Map();
  }

  push(text) {
    this.pending += text;
    const lines = this.pending.split("\n");
    this.pending = lines.pop();
    let changed = false;
    for (const rawLine of lines) {
      const values = parsePlotLine(rawLine.replace(/\r$/, ""));
      if (!values.length) continue;
      this.sample++;
      for (const { name, value } of values) {
        let points = this.series.get(name);
        if (!points) {
          points = [];
          this.series.set(name, points);
        }
        points.push({ sample: this.sample, value });
        if (points.length > this.capacity) points.splice(0, points.length - this.capacity);
      }
      changed = true;
    }
    if (changed) this.onChange(this.snapshot());
    return changed;
  }

  clear() {
    this.pending = "";
    this.sample = 0;
    this.series.clear();
    this.onChange(this.snapshot());
  }

  snapshot() {
    return {
      sample: this.sample,
      series: Array.from(this.series, ([name, points]) => ({ name, points: points.slice() })),
    };
  }
}

const colors = ["#a6be82", "#e36388", "#afc29d", "#d79bc8", "#d0b06e", "#78a887", "#bea66c", "#dba7b8"];

export class SerialPlotterView {
  constructor(canvas, legend) {
    this.canvas = canvas;
    this.legend = legend;
    this.data = { sample: 0, series: [] };
    this.resizeObserver = new ResizeObserver(() => this.render());
    this.resizeObserver.observe(canvas);
  }

  update(data) {
    this.data = data;
    this.renderLegend();
    this.render();
  }

  renderLegend() {
    this.legend.replaceChildren();
    if (!this.data.series.length) {
      const message = document.createElement("span");
      message.className = "plotter-waiting";
      message.textContent = "Waiting for Arduino-style label:value serial data…";
      this.legend.append(message);
      return;
    }
    this.data.series.forEach((series, index) => {
      const item = document.createElement("span");
      item.className = "plotter-legend-item";
      const swatch = document.createElement("i");
      swatch.style.background = colors[index % colors.length];
      const value = document.createElement("strong");
      value.textContent = `${series.name} ${formatValue(series.points.at(-1)?.value)}`;
      item.append(swatch, value);
      this.legend.append(item);
    });
  }

  render() {
    const rect = this.canvas.getBoundingClientRect();
    const scale = globalThis.devicePixelRatio || 1;
    const width = Math.max(1, Math.round(rect.width * scale));
    const height = Math.max(1, Math.round(rect.height * scale));
    if (this.canvas.width !== width || this.canvas.height !== height) {
      this.canvas.width = width;
      this.canvas.height = height;
    }
    const context = this.canvas.getContext("2d");
    context.setTransform(scale, 0, 0, scale, 0, 0);
    context.clearRect(0, 0, rect.width, rect.height);
    context.fillStyle = "#12010f";
    context.fillRect(0, 0, rect.width, rect.height);
    if (!this.data.series.length) return;

    const left = 104;
    const right = 12;
    const laneHeight = rect.height / this.data.series.length;
    const firstSample = Math.max(1, this.data.sample - 239);
    const sampleRange = Math.max(1, this.data.sample - firstSample);
    context.font = "11px ui-monospace, SFMono-Regular, Consolas, monospace";
    context.lineWidth = 1;
    this.data.series.forEach((series, index) => {
      const top = index * laneHeight;
      const bottom = top + laneHeight;
      const values = series.points.map(({ value }) => value);
      let minimum = Math.min(...values);
      let maximum = Math.max(...values);
      let padding = (maximum - minimum) * 0.08;
      if (!padding) padding = Math.max(Math.abs(maximum) * 0.02, 1);
      minimum -= padding;
      maximum += padding;

      context.strokeStyle = "#3c2235";
      context.beginPath();
      context.moveTo(left, top + laneHeight / 2);
      context.lineTo(rect.width - right, top + laneHeight / 2);
      context.stroke();
      if (index) {
        context.beginPath(); context.moveTo(0, top); context.lineTo(rect.width, top); context.stroke();
      }
      context.fillStyle = "#dccde8";
      context.fillText(series.name, 8, top + 15);
      context.fillStyle = "#a894ac";
      context.fillText(formatValue(maximum), 8, top + 30);
      context.fillText(formatValue(minimum), 8, Math.max(top + 43, bottom - 7));

      context.strokeStyle = colors[index % colors.length];
      context.lineWidth = 1.5;
      context.beginPath();
      let started = false;
      for (const point of series.points) {
        const x = left + (point.sample - firstSample) / sampleRange * Math.max(1, rect.width - left - right);
        const y = bottom - 8 - (point.value - minimum) / (maximum - minimum) * Math.max(1, laneHeight - 16);
        if (!started) { context.moveTo(x, y); started = true; } else context.lineTo(x, y);
      }
      context.stroke();
    });
  }
}

function formatValue(value) {
  if (!Number.isFinite(value)) return "—";
  const magnitude = Math.abs(value);
  if ((magnitude !== 0 && magnitude < 0.001) || magnitude >= 100000) return value.toExponential(3);
  return Number(value.toPrecision(6)).toString();
}
