import { readFileSync } from "node:fs";

const minimumPercent = 80;
const [currentPath = "coverage/coverage-final.json"] = process.argv.slice(2);

function readCoverage(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function summarize(coverage) {
  const totals = {
    statements: { covered: 0, total: 0 },
    branches: { covered: 0, total: 0 },
    functions: { covered: 0, total: 0 },
    lines: { covered: 0, total: 0 },
  };

  for (const file of Object.values(coverage)) {
    for (const count of Object.values(file.s)) {
      totals.statements.total += 1;
      totals.statements.covered += Number(count > 0);
    }

    for (const count of Object.values(file.f)) {
      totals.functions.total += 1;
      totals.functions.covered += Number(count > 0);
    }

    for (const counts of Object.values(file.b)) {
      for (const count of counts) {
        totals.branches.total += 1;
        totals.branches.covered += Number(count > 0);
      }
    }

    const lines = new Map();
    for (const [id, statement] of Object.entries(file.statementMap)) {
      const line = statement.start.line;
      lines.set(line, (lines.get(line) ?? false) || file.s[id] > 0);
    }

    for (const covered of lines.values()) {
      totals.lines.total += 1;
      totals.lines.covered += Number(covered);
    }
  }

  return totals;
}

function percentage(metric) {
  return (metric.covered / metric.total) * 100;
}

function format(metric) {
  return `${percentage(metric).toFixed(2)}% (${metric.covered}/${metric.total})`;
}

const current = summarize(readCoverage(currentPath));
let failed = false;

console.log("Coverage gates:");
for (const [name, metric] of Object.entries(current)) {
  const meetsThreshold = metric.covered * 100 >= minimumPercent * metric.total;

  console.log(
    `- ${name}: ${format(metric)} (minimum ${minimumPercent.toFixed(2)}%)`,
  );

  if (!meetsThreshold) {
    console.error(`${name} coverage is below ${minimumPercent}%.`);
    failed = true;
  }
}

if (failed) {
  process.exitCode = 1;
}
