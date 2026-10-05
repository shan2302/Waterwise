import { useCallback, useEffect, useMemo, useState } from 'react'
import UsageChart from './components/UsageChart.jsx'
import LocationTrendChart from './components/LocationTrendChart.jsx'
import { publicLocationTrends, unitDescriptions } from './data/publicLocationTrends.js'
import { api } from './services/api.js'

const today = new Date().toISOString().slice(0, 10)
const emptyUsage = { usageDate: today, consumptionLitres: '', temperature: '', rainfall: '', occupancy: '' }
const emptyPrediction = { predictionDate: today, temperature: '', rainfall: '', occupancy: '' }
const number = (value) => Number(value || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })

function App() {
  const [section, setSection] = useState('overview')
  const [selectedLocationId, setSelectedLocationId] = useState('muzaffarpur')
  const [records, setRecords] = useState([])
  const [summary, setSummary] = useState(null)
  const [predictions, setPredictions] = useState([])
  const [latest, setLatest] = useState(null)
  const [usageForm, setUsageForm] = useState(emptyUsage)
  const [predictionForm, setPredictionForm] = useState(emptyPrediction)
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const refresh = useCallback(async () => {
    setError('')
    try {
      const [usageData, summaryData, predictionData] = await Promise.all([api.usage(), api.summary(), api.predictions()])
      setRecords(usageData)
      setSummary(summaryData)
      setPredictions(predictionData)
      setLatest(predictionData[0] || null)
    } catch (requestError) {
      setError(`${requestError.message}. Check that MySQL and the Spring Boot backend are running.`)
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { refresh() }, [refresh])
  const chartRecords = useMemo(() => records.slice(-14), [records])
  const selectedLocation = publicLocationTrends.find((item) => item.id === selectedLocationId)
  const locationValues = selectedLocation?.records.map((item) => item.value) || []
  const locationAverage = locationValues.length ? locationValues.reduce((sum, value) => sum + value, 0) / locationValues.length : 0
  const latestLocationValue = selectedLocation?.records.at(-1)

  async function submitUsage(event) {
    event.preventDefault(); setError(''); setNotice(''); setSaving(true)
    try {
      await api.addUsage({ ...usageForm, consumptionLitres: Number(usageForm.consumptionLitres), temperature: Number(usageForm.temperature), rainfall: Number(usageForm.rainfall), occupancy: Number(usageForm.occupancy) })
      setUsageForm(emptyUsage); setNotice('Water usage record added.'); await refresh()
    } catch (requestError) { setError(requestError.message) } finally { setSaving(false) }
  }

  async function submitPrediction(event) {
    event.preventDefault(); setError(''); setNotice(''); setSaving(true)
    try {
      const result = await api.predict({ ...predictionForm, temperature: Number(predictionForm.temperature), rainfall: Number(predictionForm.rainfall), occupancy: Number(predictionForm.occupancy) })
      setLatest(result); setPredictions((items) => [result, ...items]); setNotice('Prediction generated from the available historical records.')
    } catch (requestError) { setError(requestError.message) } finally { setSaving(false) }
  }

  const summaryCards = selectedLocationId === 'local'
    ? [
      ['Total consumption', `${number(summary?.totalConsumption)} L`, 'Across saved building records'],
      ['Average per day', `${number(summary?.averageConsumption)} L`, 'Saved daily average'],
      ['Highest day', `${number(summary?.highestConsumption)} L`, 'Highest saved daily use'],
      ['Lowest day', `${number(summary?.lowestConsumption)} L`, 'Lowest saved daily use'],
    ]
      : selectedLocation.id === 'muzaffarpur'
      ? [
        ['Latest reported', `${number(latestLocationValue?.value)} ${selectedLocation.unit}`, '2017 system capacity'],
        ['Published records', String(locationValues.length), '2009 supply and 2015/2017 capacity'],
        ['2009 reported supply', `${number(selectedLocation.records[0].value)} ${selectedLocation.unit}`, 'Municipal water works figure'],
        ['Later reported capacity', `${number(selectedLocation.records[1].value)} ${selectedLocation.unit}`, 'AMRUT planning baseline'],
      ]
      : selectedLocation.id === 'srinagar'
      ? [
        ['Latest rural coverage', `${number(latestLocationValue?.value)}%`, 'Households reported with a tap connection'],
        ['Dated reports', String(locationValues.length), '2020, 2021, and 2023'],
        ['Earliest → latest', `${number(selectedLocation.records[0].value)}% → ${number(latestLocationValue?.value)}%`, 'Reported connection coverage'],
        ['Indicator', 'Tap access', 'Not water volume or consumption'],
      ]
      : selectedLocation.id === 'india'
      ? [
        ['Latest annual availability', `${number(latestLocationValue?.value)} ${selectedLocation.unit}`, latestLocationValue?.period || ''],
        ['Average reported', `${number(locationAverage)} ${selectedLocation.unit}`, `${locationValues.length} published estimates`],
        ['Highest estimate', `${number(Math.max(...locationValues))} ${selectedLocation.unit}`, selectedLocation.indicator],
        ['Lowest estimate', `${number(Math.min(...locationValues))} ${selectedLocation.unit}`, selectedLocation.indicator],
      ]
      : selectedLocation.records.length === 1
        ? [
          ['Published value', `${number(latestLocationValue?.value)} ${selectedLocation.unit}`, `${latestLocationValue?.period || ''} · ${selectedLocation.indicator}`],
          ['Available records', '1', 'Only one value is published for this location'],
          ['Trend over time', 'Unavailable', 'A second reporting period is needed'],
          ['Daily consumption', 'Not reported', 'The source reports a different indicator'],
        ]
        : [
          ['Latest reported', `${number(latestLocationValue?.value)} ${selectedLocation.unit}`, `${latestLocationValue?.period || ''} · ${selectedLocation.indicator}`],
          ['Average reported', `${number(locationAverage)} ${selectedLocation.unit}`, `${locationValues.length} published values`],
          ['Highest reported', `${number(Math.max(...locationValues))} ${selectedLocation.unit}`, 'Within the published records'],
          ['Lowest reported', `${number(Math.min(...locationValues))} ${selectedLocation.unit}`, 'Within the published records'],
        ]

  return <div className="app-shell">
    <aside className="sidebar">
      <a className="brand" href="#overview" onClick={() => setSection('overview')}><span className="brand-mark">W</span><span><strong>Waterwise</strong><small>Demand & conservation</small></span></a>
      <div className="nav-label">WORKSPACE</div>
      <nav aria-label="Main navigation">
        {[['overview', 'Overview', '◫'], ['usage', 'Water usage', '▤'], ['prediction', 'Prediction', '⌁']].map(([id, label, icon]) => <button className={`nav-item ${section === id ? 'active' : ''}`} key={id} onClick={() => { setSection(id); setError(''); setNotice('') }}><span>{icon}</span>{label}</button>)}
      </nav>
      <div className="sidebar-note"><div className="note-icon">♧</div><strong>Every drop counts</strong><p>Better planning starts with understanding when water is needed.</p></div>
      <div className="sidebar-footer">BCS508 · Environmental Studies<br/>Local prototype · ₹0 cost</div>
    </aside>

    <main className="main-content">
      <header className="topbar"><div className="breadcrumbs">Workspace <span>/</span> {section === 'overview' ? 'Overview' : section === 'usage' ? 'Water usage' : 'Prediction'}</div><div className="local-pill"><i /> Local system</div></header>
      <div className="content">
        {error && <div className="alert error" role="alert">{error}<button onClick={() => setError('')}>×</button></div>}
        {notice && <div className="alert success" role="status">{notice}<button onClick={() => setNotice('')}>×</button></div>}
        {loading ? <div className="loading-card">Connecting to the local water demand system…</div> : <>
          {section === 'overview' && <>
            <div className="page-heading"><div><div className="eyebrow">WATER RESOURCE MONITORING</div><h1>Overview</h1><p>Choose a place to view its published water data and trend.</p></div><button className="primary-button" onClick={() => setSection('prediction')}>Generate prediction <span>→</span></button></div>
            <LocationSelector value={selectedLocationId} onChange={setSelectedLocationId} description="View overview for" />
            <div className="stat-grid">{summaryCards.map(([label, value, helper], index) => <article className="stat-card" key={label}><div className="stat-label">{label}<span className="stat-glyph">{['◉', '∿', '↗', '↘'][index]}</span></div><div className="stat-value">{value}</div><div className="stat-helper">{helper}</div></article>)}</div>
            {selectedLocationId !== 'local' && <PlanningEstimate location={selectedLocation} />}
            <div className="overview-grid"><section className="panel chart-panel"><div className="panel-heading"><div><h2>{selectedLocationId === 'local' ? 'Historical water consumption' : `${selectedLocation.name}: published water data`}</h2><p>{selectedLocationId === 'local' ? 'Daily use across the most recent 14 records' : `${selectedLocation.indicator}${selectedLocation.records.length === 1 ? ' · One published value; no time trend is available' : ''}`}</p></div><span className="small-tag">{selectedLocationId === 'local' ? 'LITRES' : selectedLocation.unit}</span></div>{selectedLocationId === 'local' ? <UsageChart records={chartRecords}/> : <LocationTrendChart location={selectedLocation}/>}</section>
              {selectedLocationId === 'local' ? <section className="panel prediction-panel"><div className="panel-heading"><div><h2>Latest prediction</h2><p>Estimate from local building records</p></div><span className="water-icon">⌁</span></div>{latest ? <><div className="latest-date">{latest.predictionDate}</div><div className="latest-value">{number(latest.predictedDemandLitres)} <span>L</span></div><span className={`status-badge ${latest.demandStatus.toLowerCase()}`}>{latest.demandStatus} DEMAND</span><p className="recommendation">{latest.recommendation}</p></> : <div className="empty-state"><p>No prediction yet.</p><button className="text-button" onClick={() => setSection('prediction')}>Create your first prediction →</button></div>}</section> : <section className="panel prediction-panel"><div className="panel-heading"><div><h2>Prediction scope</h2><p>How this location data is used</p></div><span className="water-icon">i</span></div><p className="recommendation">This chart summarizes published location data. It is not used as building-level consumption. Demand predictions use records saved in the local prototype.</p><button className="text-button" onClick={() => setSelectedLocationId('local')}>View local prototype data →</button></section>}
            </div>
            {selectedLocationId !== 'local' && <section className="panel city-data-panel"><div className="panel-heading"><div><div className="eyebrow">{selectedLocation.name.toUpperCase()} · {selectedLocation.region.toUpperCase()}</div><h2>{selectedLocation.indicator}</h2><p>{selectedLocation.note}</p></div><span className="small-tag">PUBLIC SOURCE</span></div><p className="city-data-note">Source: <a className="source-link" href={selectedLocation.sourceUrl} target="_blank" rel="noreferrer">{selectedLocation.sourceLabel} ↗</a></p><p className="city-data-note">These published values use the source's reporting periods. The prediction form continues to use the local prototype's building records.</p></section>}
            {selectedLocationId === 'local' && <section className="panel recent-panel"><div className="panel-heading"><div><h2>Recent usage records</h2><p>Latest saved observations</p></div><button className="text-button" onClick={() => setSection('usage')}>View all records →</button></div><UsageTable records={records.slice(-5).reverse()}/></section>}
          </>}
          {section === 'usage' && <><div className="page-heading"><div><div className="eyebrow">HISTORICAL DATA</div><h1>Water usage</h1><p>Browse location reports or manage local building records.</p></div><div className="record-count">{selectedLocationId === 'local' ? `${records.length} records` : selectedLocation.name}</div></div>
            <LocationSelector value={selectedLocationId} onChange={setSelectedLocationId} description="View usage data for" />
            {selectedLocationId === 'local' ? <><div className="usage-layout"><section className="panel form-panel"><div className="panel-heading"><div><h2>Add a usage record</h2><p>Enter the daily information below.</p></div></div><form onSubmit={submitUsage} className="form-grid">
              <Field label="Date"><input type="date" required value={usageForm.usageDate} onChange={(e) => setUsageForm({ ...usageForm, usageDate: e.target.value })}/></Field>
              <Field label="Consumption (litres)"><input type="number" required min="0.01" step="any" placeholder="e.g. 5200" value={usageForm.consumptionLitres} onChange={(e) => setUsageForm({ ...usageForm, consumptionLitres: e.target.value })}/></Field>
              <Field label="Temperature (°C)"><input type="number" required min="-20" max="60" step="any" placeholder="e.g. 29" value={usageForm.temperature} onChange={(e) => setUsageForm({ ...usageForm, temperature: e.target.value })}/></Field>
              <Field label="Rainfall (mm)"><input type="number" required min="0" max="1000" step="any" placeholder="e.g. 2" value={usageForm.rainfall} onChange={(e) => setUsageForm({ ...usageForm, rainfall: e.target.value })}/></Field>
              <Field label="Occupancy (people)"><input type="number" required min="1" max="1000000" step="1" placeholder="e.g. 100" value={usageForm.occupancy} onChange={(e) => setUsageForm({ ...usageForm, occupancy: e.target.value })}/></Field>
              <div className="form-actions"><button className="primary-button" disabled={saving}>{saving ? 'Saving…' : 'Save record'}</button></div>
            </form></section><section className="panel data-note"><div className="note-icon">i</div><h3>About this data</h3><p>Sample records are included for demonstration. They are illustrative values, not physical measurements collected by the project team.</p><p>This prototype uses no sensors or external services.</p></section></div>
            <section className="panel recent-panel"><div className="panel-heading"><div><h2>All usage records</h2><p>Records saved in the local MySQL database</p></div></div><UsageTable records={[...records].reverse()} showWeather/></section></> : <PublicLocationView location={selectedLocation} />}
          </>}
          {section === 'prediction' && <><div className="page-heading"><div><div className="eyebrow">DEMAND ESTIMATION</div><h1>Water demand prediction</h1><p>Estimate expected use with historical records and environmental inputs.</p></div></div>
            <LocationSelector value={selectedLocationId} onChange={setSelectedLocationId} description="Prediction data source" />
            {selectedLocationId === 'local' ? <><div className="prediction-layout"><section className="panel form-panel"><div className="panel-heading"><div><h2>Prediction inputs</h2><p>Provide expected conditions for the selected date.</p></div></div><form onSubmit={submitPrediction} className="form-grid">
              <Field label="Prediction date"><input type="date" required value={predictionForm.predictionDate} onChange={(e) => setPredictionForm({ ...predictionForm, predictionDate: e.target.value })}/></Field>
              <Field label="Temperature (°C)"><input type="number" required min="-20" max="60" step="any" placeholder="e.g. 29" value={predictionForm.temperature} onChange={(e) => setPredictionForm({ ...predictionForm, temperature: e.target.value })}/></Field>
              <Field label="Rainfall (mm)"><input type="number" required min="0" max="1000" step="any" placeholder="e.g. 2" value={predictionForm.rainfall} onChange={(e) => setPredictionForm({ ...predictionForm, rainfall: e.target.value })}/></Field>
              <Field label="Occupancy (people)"><input type="number" required min="1" max="1000000" step="1" placeholder="e.g. 100" value={predictionForm.occupancy} onChange={(e) => setPredictionForm({ ...predictionForm, occupancy: e.target.value })}/></Field>
              <div className="form-actions"><button className="primary-button" disabled={saving}>{saving ? 'Calculating…' : 'Generate prediction'}</button></div>
            </form><div className="model-note"><span>⌁</span><p><strong>How it works</strong><br/>A simple regression model uses saved historical records. Estimates are compared with the historical average.</p></div></section>
              <section className="panel result-panel"><div className="panel-heading"><div><h2>Prediction result</h2><p>{latest ? `For ${latest.predictionDate}` : 'Your result will appear here'}</p></div></div>{latest ? <><div className="result-number">{number(latest.predictedDemandLitres)} <span>litres</span></div><div className="result-status-row"><span>Demand status</span><span className={`status-badge ${latest.demandStatus.toLowerCase()}`}>{latest.demandStatus}</span></div><div className="advice-box"><strong>Conservation recommendation</strong><p>{latest.recommendation}</p></div><div className="threshold-note">HIGH means the estimate is over 10% above the historical average.</div></> : <div className="empty-result"><span>⌁</span><p>Enter the expected conditions and generate a prediction to see the estimated demand and conservation guidance.</p></div>}</section></div>
            <section className="panel recent-panel"><div className="panel-heading"><div><h2>Previous predictions</h2><p>Saved prediction history</p></div></div>{predictions.length ? <div className="table-scroll"><table><thead><tr><th>Prediction date</th><th>Estimated demand</th><th>Status</th><th>Generated</th></tr></thead><tbody>{predictions.map((item) => <tr key={item.id}><td>{item.predictionDate}</td><td>{number(item.predictedDemandLitres)} L</td><td><span className={`status-badge ${item.demandStatus.toLowerCase()}`}>{item.demandStatus}</span></td><td>{new Date(item.createdAt).toLocaleString()}</td></tr>)}</tbody></table></div> : <p className="table-empty">Predictions will appear here after generation.</p>}</section></> : <><PlanningEstimate location={selectedLocation} /><section className="public-prediction-view panel"><div className="eyebrow">CITY DAILY PLANNING ESTIMATE</div><h2>{selectedLocation.name}: reference demand only</h2><p>This population-based planning estimate is not a measured daily consumption value or an AI/ML prediction. The public records do not provide the matching daily consumption, weather, rainfall, and occupancy history required by the model. Choose local prototype records to generate an AI/ML estimate from the saved building data.</p><button className="primary-button" onClick={() => setSelectedLocationId('local')}>Use local prototype records</button></section></>}
          </>}
        </>}
        <footer className="page-footer">AI-Based Water Demand Prediction and Conservation System <span>•</span> BCS508 Semester V</footer>
      </div>
    </main>
  </div>
}

function Field({ label, children }) { return <label className="field"><span>{label}</span>{children}</label> }

function PlanningEstimate({ location }) {
  const litresPerDay = location.planningPopulation * 135
  const mld = litresPerDay / 1_000_000
  return <section className="panel planning-estimate"><div><div className="eyebrow">{location.name.toUpperCase()} · PLANNING REFERENCE</div><h2>{number(mld)} MLD</h2><p>{number(litresPerDay)} litres per day</p></div><div className="planning-estimate-details"><strong>Estimated domestic demand baseline</strong><span>Formula: {number(location.planningPopulation)} people × 135 litres/person/day.</span><span>Population: {location.planningPopulationLabel}. Population boundaries can differ between locations.</span><span>This is a planning estimate, not measured consumption, actual supply, or an AI/ML forecast.</span><span>Sources: <a className="source-link" href={location.planningPopulationSourceUrl} target="_blank" rel="noreferrer">{location.planningPopulationSource} ↗</a> · <a className="source-link" href="https://cpheeo.gov.in/upload/uploadfiles/files/Handbook.pdf" target="_blank" rel="noreferrer">CPHEEO 135 LPCD benchmark ↗</a></span></div></section>
}

function LocationSelector({ value, onChange, description }) {
  const selected = publicLocationTrends.find((location) => location.id === value)
  const unitMeaning = selected && unitDescriptions[selected.unit]
  return <section className="location-filter panel"><label htmlFor="location-select"><strong>{description}</strong><span className="field-hint">{selected ? `Unit: ${selected.unit}${unitMeaning ? ` means ${unitMeaning}` : ''}.` : 'Choose a location or your local prototype records.'}</span></label><select id="location-select" value={value} onChange={(event) => onChange(event.target.value)}><option value="local">My local prototype records</option>{publicLocationTrends.map((location) => <option value={location.id} key={location.id}>{location.name} — {location.region}</option>)}</select></section>
}

function PublicLocationView({ location }) {
  return <div className="public-location-view">
    <PlanningEstimate location={location} />
    <section className="panel public-location-chart"><div className="panel-heading"><div><div className="eyebrow">{location.name.toUpperCase()} · {location.region.toUpperCase()}</div><h2>{location.indicator}</h2><p>{location.records.length} published records · {location.unit}{unitDescriptions[location.unit] ? ` (${unitDescriptions[location.unit]})` : ''}</p></div><span className="small-tag">PUBLIC DATA</span></div><LocationTrendChart location={location}/><p className="public-location-note">{location.note}</p></section>
    <section className="panel recent-panel"><div className="panel-heading"><div><h2>Published records</h2><p>Each row links to the source used for that value.</p></div></div><div className="table-scroll"><table><thead><tr><th>Period</th><th>Indicator</th><th>Reported value</th><th>Source</th></tr></thead><tbody>{location.records.map((record) => <tr key={record.period}><td>{record.period}</td><td>{record.note || location.indicator}</td><td className="table-emphasis">{number(record.value)} {location.unit}</td><td><a className="source-link" href={record.sourceUrl || location.sourceUrl} target="_blank" rel="noreferrer">{record.sourceLabel || location.sourceLabel} ↗</a></td></tr>)}</tbody></table></div></section>
  </div>
}

function UsageTable({ records, showWeather = false }) {
  if (!records.length) return <p className="table-empty">No water usage records are available.</p>
  return <div className="table-scroll"><table><thead><tr><th>Date</th><th>Consumption</th>{showWeather && <><th>Temperature</th><th>Rainfall</th><th>Occupancy</th></>}</tr></thead><tbody>{records.map((record) => <tr key={record.id}><td>{record.usageDate}</td><td className="table-emphasis">{number(record.consumptionLitres)} L</td>{showWeather && <><td>{number(record.temperature)} °C</td><td>{number(record.rainfall)} mm</td><td>{number(record.occupancy)}</td></>}</tr>)}</tbody></table></div>
}

export default App
