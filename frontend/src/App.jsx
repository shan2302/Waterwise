import { useCallback, useEffect, useMemo, useState } from 'react'
import UsageChart from './components/UsageChart.jsx'
import { api } from './services/api.js'

const today = new Date().toISOString().slice(0, 10)
const emptyUsage = { usageDate: today, consumptionLitres: '', temperature: '', rainfall: '', occupancy: '' }
const emptyPrediction = { predictionDate: today, temperature: '', rainfall: '', occupancy: '' }
const number = (value) => Number(value || 0).toLocaleString(undefined, { maximumFractionDigits: 1 })

function App() {
  const [section, setSection] = useState('overview')
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

  const summaryCards = [
    ['Total consumption', `${number(summary?.totalConsumption)} L`, 'Across recorded days'],
    ['Average per day', `${number(summary?.averageConsumption)} L`, 'Recorded daily average'],
    ['Highest day', `${number(summary?.highestConsumption)} L`, 'Maximum recorded use'],
    ['Lowest day', `${number(summary?.lowestConsumption)} L`, 'Minimum recorded use'],
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
            <div className="page-heading"><div><div className="eyebrow">WATER RESOURCE MONITORING</div><h1>Overview</h1><p>A clear view of recent water use and expected demand.</p></div><button className="primary-button" onClick={() => setSection('prediction')}>Generate prediction <span>→</span></button></div>
            <div className="stat-grid">{summaryCards.map(([label, value, helper]) => <article className="stat-card" key={label}><div className="stat-label">{label}<span className="stat-glyph">{label === 'Total consumption' ? '◉' : label === 'Average per day' ? '∿' : label === 'Highest day' ? '↗' : '↘'}</span></div><div className="stat-value">{value}</div><div className="stat-helper">{helper}</div></article>)}</div>
            <div className="overview-grid"><section className="panel chart-panel"><div className="panel-heading"><div><h2>Historical water consumption</h2><p>Daily use across the most recent 14 records</p></div><span className="small-tag">LITRES</span></div><UsageChart records={chartRecords}/></section>
              <section className="panel prediction-panel"><div className="panel-heading"><div><h2>Latest prediction</h2><p>Most recent demand estimate</p></div><span className="water-icon">⌁</span></div>{latest ? <><div className="latest-date">{latest.predictionDate}</div><div className="latest-value">{number(latest.predictedDemandLitres)} <span>L</span></div><span className={`status-badge ${latest.demandStatus.toLowerCase()}`}>{latest.demandStatus} DEMAND</span><p className="recommendation">{latest.recommendation}</p></> : <div className="empty-state"><p>No prediction yet.</p><button className="text-button" onClick={() => setSection('prediction')}>Create your first prediction →</button></div>}</section>
            </div>
            <section className="panel city-data-panel"><div className="panel-heading"><div><div className="eyebrow">PUBLIC CITY DATA · BENGALURU, KARNATAKA</div><h2>Water supply context</h2><p>Published figures from the BWSSB Annual Report 2020–21.</p></div><span className="small-tag">REAL LOCATION</span></div><div className="city-data-grid"><article><strong>1,227 MLD</strong><span>Average water received by BWSSB during 2020–21</span></article><article><strong>1,445 MLD</strong><span>Designed treated-water capacity across the Cauvery Water Supply Scheme stages</span></article></div><p className="city-data-note">MLD means million litres per day. These are city-wide utility figures, not building-level consumption, so they are shown for local context and are not used to train the prototype prediction.</p><a className="source-link" href="https://kla.kar.nic.in/council/house/Paperlaid/147/91.pdf" target="_blank" rel="noreferrer">Source: Bangalore Water Supply and Sewerage Board Annual Report 2020–21 (Karnataka Legislative Council) ↗</a></section>
            <section className="panel recent-panel"><div className="panel-heading"><div><h2>Recent usage records</h2><p>Latest saved observations</p></div><button className="text-button" onClick={() => setSection('usage')}>View all records →</button></div><UsageTable records={records.slice(-5).reverse()}/></section>
          </>}
          {section === 'usage' && <><div className="page-heading"><div><div className="eyebrow">HISTORICAL DATA</div><h1>Water usage</h1><p>Review records or add an observation to the local dataset.</p></div><div className="record-count">{records.length} records</div></div>
            <div className="usage-layout"><section className="panel form-panel"><div className="panel-heading"><div><h2>Add a usage record</h2><p>Enter the daily information below.</p></div></div><form onSubmit={submitUsage} className="form-grid">
              <Field label="Date"><input type="date" required value={usageForm.usageDate} onChange={(e) => setUsageForm({ ...usageForm, usageDate: e.target.value })}/></Field>
              <Field label="Consumption (litres)"><input type="number" required min="0.01" step="any" placeholder="e.g. 5200" value={usageForm.consumptionLitres} onChange={(e) => setUsageForm({ ...usageForm, consumptionLitres: e.target.value })}/></Field>
              <Field label="Temperature (°C)"><input type="number" required min="-20" max="60" step="any" placeholder="e.g. 29" value={usageForm.temperature} onChange={(e) => setUsageForm({ ...usageForm, temperature: e.target.value })}/></Field>
              <Field label="Rainfall (mm)"><input type="number" required min="0" max="1000" step="any" placeholder="e.g. 2" value={usageForm.rainfall} onChange={(e) => setUsageForm({ ...usageForm, rainfall: e.target.value })}/></Field>
              <Field label="Occupancy (people)"><input type="number" required min="1" max="1000000" step="1" placeholder="e.g. 100" value={usageForm.occupancy} onChange={(e) => setUsageForm({ ...usageForm, occupancy: e.target.value })}/></Field>
              <div className="form-actions"><button className="primary-button" disabled={saving}>{saving ? 'Saving…' : 'Save record'}</button></div>
            </form></section><section className="panel data-note"><div className="note-icon">i</div><h3>About this data</h3><p>Sample records are included for demonstration. They are illustrative values, not physical measurements collected by the project team.</p><p>This prototype uses no sensors or external services.</p></section></div>
            <section className="panel recent-panel"><div className="panel-heading"><div><h2>All usage records</h2><p>Records saved in the local MySQL database</p></div></div><UsageTable records={[...records].reverse()} showWeather/></section>
          </>}
          {section === 'prediction' && <><div className="page-heading"><div><div className="eyebrow">DEMAND ESTIMATION</div><h1>Water demand prediction</h1><p>Estimate expected use with historical records and environmental inputs.</p></div></div>
            <div className="prediction-layout"><section className="panel form-panel"><div className="panel-heading"><div><h2>Prediction inputs</h2><p>Provide expected conditions for the selected date.</p></div></div><form onSubmit={submitPrediction} className="form-grid">
              <Field label="Prediction date"><input type="date" required value={predictionForm.predictionDate} onChange={(e) => setPredictionForm({ ...predictionForm, predictionDate: e.target.value })}/></Field>
              <Field label="Temperature (°C)"><input type="number" required min="-20" max="60" step="any" placeholder="e.g. 29" value={predictionForm.temperature} onChange={(e) => setPredictionForm({ ...predictionForm, temperature: e.target.value })}/></Field>
              <Field label="Rainfall (mm)"><input type="number" required min="0" max="1000" step="any" placeholder="e.g. 2" value={predictionForm.rainfall} onChange={(e) => setPredictionForm({ ...predictionForm, rainfall: e.target.value })}/></Field>
              <Field label="Occupancy (people)"><input type="number" required min="1" max="1000000" step="1" placeholder="e.g. 100" value={predictionForm.occupancy} onChange={(e) => setPredictionForm({ ...predictionForm, occupancy: e.target.value })}/></Field>
              <div className="form-actions"><button className="primary-button" disabled={saving}>{saving ? 'Calculating…' : 'Generate prediction'}</button></div>
            </form><div className="model-note"><span>⌁</span><p><strong>How it works</strong><br/>A simple regression model uses saved historical records. Estimates are compared with the historical average.</p></div></section>
              <section className="panel result-panel"><div className="panel-heading"><div><h2>Prediction result</h2><p>{latest ? `For ${latest.predictionDate}` : 'Your result will appear here'}</p></div></div>{latest ? <><div className="result-number">{number(latest.predictedDemandLitres)} <span>litres</span></div><div className="result-status-row"><span>Demand status</span><span className={`status-badge ${latest.demandStatus.toLowerCase()}`}>{latest.demandStatus}</span></div><div className="advice-box"><strong>Conservation recommendation</strong><p>{latest.recommendation}</p></div><div className="threshold-note">HIGH means the estimate is over 10% above the historical average.</div></> : <div className="empty-result"><span>⌁</span><p>Enter the expected conditions and generate a prediction to see the estimated demand and conservation guidance.</p></div>}</section></div>
            <section className="panel recent-panel"><div className="panel-heading"><div><h2>Previous predictions</h2><p>Saved prediction history</p></div></div>{predictions.length ? <div className="table-scroll"><table><thead><tr><th>Prediction date</th><th>Estimated demand</th><th>Status</th><th>Generated</th></tr></thead><tbody>{predictions.map((item) => <tr key={item.id}><td>{item.predictionDate}</td><td>{number(item.predictedDemandLitres)} L</td><td><span className={`status-badge ${item.demandStatus.toLowerCase()}`}>{item.demandStatus}</span></td><td>{new Date(item.createdAt).toLocaleString()}</td></tr>)}</tbody></table></div> : <p className="table-empty">Predictions will appear here after generation.</p>}</section>
          </>}
        </>}
        <footer className="page-footer">AI-Based Water Demand Prediction and Conservation System <span>•</span> BCS508 Semester V</footer>
      </div>
    </main>
  </div>
}

function Field({ label, children }) { return <label className="field"><span>{label}</span>{children}</label> }

function UsageTable({ records, showWeather = false }) {
  if (!records.length) return <p className="table-empty">No water usage records are available.</p>
  return <div className="table-scroll"><table><thead><tr><th>Date</th><th>Consumption</th>{showWeather && <><th>Temperature</th><th>Rainfall</th><th>Occupancy</th></>}</tr></thead><tbody>{records.map((record) => <tr key={record.id}><td>{record.usageDate}</td><td className="table-emphasis">{number(record.consumptionLitres)} L</td>{showWeather && <><td>{number(record.temperature)} °C</td><td>{number(record.rainfall)} mm</td><td>{number(record.occupancy)}</td></>}</tr>)}</tbody></table></div>
}

export default App
