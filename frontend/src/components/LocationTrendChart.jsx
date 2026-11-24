function LocationTrendChart({ location }) {
  const records = location.records
  const width = 760
  const height = 230
  const pad = { left: 58, right: 18, top: 20, bottom: 36 }
  const values = records.map((item) => item.value)
  const low = Math.min(...values)
  const high = Math.max(...values)
  const range = Math.max(high - low, Math.abs(high) * 0.12, 1)
  const min = Math.max(0, low - range * 0.1)
  const max = high + range * 0.1
  const plotWidth = width - pad.left - pad.right
  const plotHeight = height - pad.top - pad.bottom
  const point = (item, index) => ({
    x: pad.left + (records.length === 1 ? plotWidth / 2 : index * plotWidth / (records.length - 1)),
    y: height - pad.bottom - ((item.value - min) / (max - min)) * plotHeight,
  })
  const points = records.map(point)
  const ticks = [0, 1, 2, 3].map((index) => min + (max - min) * index / 3)

  return <div className="chart-wrap">
    <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={`${location.indicator} for ${location.name}`}>
      {ticks.map((tick, index) => {
        const y = height - pad.bottom - index * plotHeight / 3
        return <g key={index}><line x1={pad.left} y1={y} x2={width - pad.right} y2={y} className="grid-line"/><text x={pad.left - 8} y={y + 4} textAnchor="end" className="axis-text">{tick.toLocaleString(undefined, { maximumFractionDigits: 1 })}</text></g>
      })}
      {points.length > 1 && <polyline points={points.map(({ x, y }) => `${x},${y}`).join(' ')} className="chart-line" fill="none"/>}
      {records.map((item, index) => <g key={item.period}><circle cx={points[index].x} cy={points[index].y} r="4" className="chart-dot"><title>{item.period}: {item.value.toLocaleString()} {location.unit}</title></circle><text x={points[index].x} y={height - 10} textAnchor="middle" className="axis-text">{item.period}</text></g>)}
    </svg>
    <div className="chart-key"><span />{location.indicator} ({location.unit})</div>
  </div>
}

export default LocationTrendChart
