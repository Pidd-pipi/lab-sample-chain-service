async function loadSamples() {
  const response = await fetch('/api/samples');
  const data = await response.json();
  document.querySelector('#samples').innerHTML = data.samples.map((sample) =>
    `<li>${sample.id} · ${sample.material} · ${sample.batch} · ${sample.status}</li>`).join('');
}
document.querySelector('#refresh').addEventListener('click', loadSamples);
loadSamples();
