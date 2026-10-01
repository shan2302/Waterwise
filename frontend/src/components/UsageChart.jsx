function UsageChart({ records }) {
  const width = 760
  const height = 230
  const pad = { left: 48, right: 15, top: 20, bottom: 34 }
  const values = records.map((item) => item.consumptionLitres)
  const min = Math.min(...values, 0)
  const max = Math.max(...values, 1)
  const range = Math.max(max - min, 1)
  const points = records.map((item, index) => {
    const x = pad.left + (records.length === 1 ? 0 : index * (width - pad.left - pad.right) / (records.length - 1))
    const y = height - pad.bottom - ((item.consumptionLitres - min) / range) * (height - pad.top - pad.bottom)
    return `${x},${y}`
  }).join(' ')
  const ticks = [0, 1, 2, 3].map((index) => Math.round(min + range * index / 3))
  return <div className="chart-wrap">
    <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Line chart of historical water consumption">
      {ticks.map((tick, index) => {
        const y = height - pad.bottom - index * (height - pad.top - pad.bottom) / 3
        return <g key={tick}><line x1={pad.left} y1={y} x2={width - pad.right} y2={y} className="grid-line"/><text x={pad.left - 8} y={y + 4} textAnchor="end" className="axis-text">{tick.toLocaleString()}</text></g>
      })}
      {records.length > 0 && <><polyline points={points} className="chart-line" fill="none"/>{records.map((item, index) => {
        const x = pad.left + (records.length === 1 ? 0 : index * (width - pad.left - pad.right) / (records.length - 1))
        const y = height - pad.bottom - ((item.consumptionLitres - min) / range) * (height - pad.top - pad.bottom)
        return <circle key={item.id} cx={x} cy={y} r="3.5" className="chart-dot"><title>{item.usageDate}: {item.consumptionLitres.toLocaleString()} L</title></circle>
      })}</>}
      <text x={pad.left} y={height - 8} className="axis-text">{records[0]?.usageDate || ''}</text>
      <text x={width - pad.right} y={height - 8} textAnchor="end" className="axis-text">{records.at(-1)?.usageDate || ''}</text>
    </svg>
    <div className="chart-key"><span /> Daily consumption (litres)</div>
  </div>
}

export default UsageChart
