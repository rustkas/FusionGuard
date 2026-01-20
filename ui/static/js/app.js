const API_BASE = window.location.origin;

let riskChart = null;
let currentShotId = null;
let autoRefreshInterval = null;
let autoRefreshEnabled = false;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadShots();
    setupEventListeners();
    initializeChart();
});

function setupEventListeners() {
    document.getElementById('shot-select').addEventListener('change', (e) => {
        currentShotId = e.target.value;
        if (currentShotId) {
            loadShotData();
        }
    });

    document.getElementById('refresh-shots').addEventListener('click', () => {
        loadShots();
    });

    document.getElementById('auto-refresh-toggle').addEventListener('click', () => {
        toggleAutoRefresh();
    });
}

function initializeChart() {
    const ctx = document.getElementById('risk-chart').getContext('2d');
    riskChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Risk H50 (50ms horizon)',
                    data: [],
                    borderColor: 'rgb(239, 68, 68)',
                    backgroundColor: 'rgba(239, 68, 68, 0.1)',
                    tension: 0.1,
                },
                {
                    label: 'Risk H200 (200ms horizon)',
                    data: [],
                    borderColor: 'rgb(245, 158, 11)',
                    backgroundColor: 'rgba(245, 158, 11, 0.1)',
                    tension: 0.1,
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 1.0,
                    title: {
                        display: true,
                        text: 'Disruption Risk'
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: 'Time'
                    }
                }
            },
            plugins: {
                legend: {
                    display: true,
                    position: 'top',
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                }
            }
        }
    });
}

async function loadShots() {
    try {
        const response = await fetch(`${API_BASE}/shots`);
        const data = await response.json();
        
        const select = document.getElementById('shot-select');
        select.innerHTML = '<option value="">Select a shot...</option>';
        
        data.shots.forEach(shot => {
            const option = document.createElement('option');
            option.value = shot.shot_id;
            option.textContent = `${shot.shot_id}${shot.started_unix_ns ? ' - ' + new Date(shot.started_unix_ns / 1_000_000).toLocaleString() : ''}`;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load shots:', error);
        alert('Failed to load shots. Make sure the API is running.');
    }
}

async function loadShotData() {
    if (!currentShotId) return;

    try {
        // Load risk series
        const riskResponse = await fetch(`${API_BASE}/shots/${currentShotId}/series?kind=risk`);
        const riskData = await riskResponse.json();
        updateRiskChart(riskData);

        // Load events
        const eventsResponse = await fetch(`${API_BASE}/shots/${currentShotId}/events`);
        const eventsData = await eventsResponse.json();
        updateEvents(eventsData.events);

        // Load latest risk point for explanations
        if (riskData.points && riskData.points.length > 0) {
            const latestPoint = riskData.points[riskData.points.length - 1];
            loadExplanation(currentShotId, latestPoint.ts_unix_ns);
        }
    } catch (error) {
        console.error('Failed to load shot data:', error);
    }
}

function updateRiskChart(riskData) {
    if (!riskChart || !riskData.points || riskData.points.length === 0) {
        return;
    }

    const points = riskData.points;
    const times = points.map(p => {
        const date = new Date(p.ts_unix_ns / 1_000_000);
        return date.toLocaleTimeString();
    });

    riskChart.data.labels = times;
    riskChart.data.datasets[0].data = points.map(p => p.risk_h50);
    riskChart.data.datasets[1].data = points.map(p => p.risk_h200);
    riskChart.update();
}

function updateEvents(events) {
    const container = document.getElementById('events');
    
    if (!events || events.length === 0) {
        container.innerHTML = '<p class="placeholder">No events for this shot</p>';
        return;
    }

    container.innerHTML = events.map(event => {
        const date = new Date(event.ts_unix_ns / 1_000_000);
        return `
            <div class="event-item ${event.kind}">
                <div class="kind">${event.kind}</div>
                <div class="message">${event.message}</div>
                <div class="timestamp">${date.toLocaleString()}</div>
            </div>
        `;
    }).join('');
}

async function loadExplanation(shotId, atUnixNs) {
    try {
        const response = await fetch(`${API_BASE}/shots/${shotId}/explain?at_unix_ns=${atUnixNs}`);
        const data = await response.json();
        
        updateTopDrivers(data.top_features || []);
    } catch (error) {
        console.error('Failed to load explanation:', error);
        updateTopDrivers([]);
    }
}

function updateTopDrivers(features) {
    const container = document.getElementById('top-drivers');
    
    if (!features || features.length === 0) {
        container.innerHTML = '<p class="placeholder">No feature data available</p>';
        return;
    }

    container.innerHTML = features.map(feature => {
        const score = feature.score || feature.value || 0;
        const name = feature.name || feature.key || 'Unknown';
        return `
            <div class="driver-item">
                <strong>${name}</strong>
                <span class="score">Contribution: ${score.toFixed(4)}</span>
            </div>
        `;
    }).join('');
}

// Placeholder for recommendations (would come from risk point data)
function updateRecommendations(recommendations) {
    const container = document.getElementById('recommendations');
    
    if (!recommendations || recommendations.length === 0) {
        container.innerHTML = '<p class="placeholder">No recommendations available</p>';
        return;
    }

    container.innerHTML = recommendations.map(rec => {
        return `
            <div class="recommendation-item">
                <div class="action">${rec.action || 'Unknown action'}</div>
                <div class="rationale">${rec.rationale || ''}</div>
            </div>
        `;
    }).join('');
}

function toggleAutoRefresh() {
    autoRefreshEnabled = !autoRefreshEnabled;
    const button = document.getElementById('auto-refresh-toggle');
    
    if (autoRefreshEnabled) {
        button.textContent = 'Auto-refresh: ON';
        button.classList.add('active');
        autoRefreshInterval = setInterval(() => {
            if (currentShotId) {
                loadShotData();
            }
        }, 5000); // Refresh every 5 seconds
    } else {
        button.textContent = 'Auto-refresh: OFF';
        button.classList.remove('active');
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
            autoRefreshInterval = null;
        }
    }
}
