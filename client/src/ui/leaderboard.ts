export interface LeaderboardEntry {
  nickname: string;
  wins: number;
  losses: number;
  draws: number;
  pointsFor: number;
  pointsAgainst: number;
  gamesPlayed: number;
}

const overlay = document.getElementById('leaderboard-overlay')!;
const content = document.getElementById('leaderboard-content')!;
const tabWins = document.getElementById('lb-tab-wins')!;
const tabPoints = document.getElementById('lb-tab-points')!;
const closeBtn = document.getElementById('leaderboard-close')!;

let currentSort: 'wins' | 'points' = 'wins';

export function showLeaderboard(): void {
  overlay.classList.remove('hidden');
  fetchLeaderboard(currentSort);
}

export function hideLeaderboard(): void {
  overlay.classList.add('hidden');
}

tabWins.addEventListener('click', () => {
  currentSort = 'wins';
  tabWins.classList.add('active');
  tabPoints.classList.remove('active');
  fetchLeaderboard('wins');
});

tabPoints.addEventListener('click', () => {
  currentSort = 'points';
  tabPoints.classList.add('active');
  tabWins.classList.remove('active');
  fetchLeaderboard('points');
});

closeBtn.addEventListener('click', hideLeaderboard);

async function fetchLeaderboard(sort: string): Promise<void> {
  content.innerHTML = '<p style="color:#94A3B8">Loading...</p>';
  try {
    const resp = await fetch(`/tournament/leaderboard?sort=${sort}`);
    const entries: LeaderboardEntry[] = await resp.json();
    renderTable(entries, sort);
  } catch {
    content.innerHTML = '<p style="color:#EF4444">Failed to load leaderboard</p>';
  }
}

function renderTable(entries: LeaderboardEntry[], sort: string): void {
  if (!entries || entries.length === 0) {
    content.innerHTML = '<p style="color:#94A3B8">No tournament games played yet</p>';
    return;
  }

  const header = sort === 'points'
    ? '<th>#</th><th>Player</th><th>PTS</th><th>W</th><th>L</th><th>D</th><th>GP</th>'
    : '<th>#</th><th>Player</th><th>W</th><th>L</th><th>D</th><th>PTS</th><th>GP</th>';

  const rows = entries.map((e, i) => {
    const rank = i + 1;
    const name = escapeHtml(e.nickname);
    if (sort === 'points') {
      return `<tr><td>${rank}</td><td>${name}</td><td>${e.pointsFor}</td><td>${e.wins}</td><td>${e.losses}</td><td>${e.draws}</td><td>${e.gamesPlayed}</td></tr>`;
    }
    return `<tr><td>${rank}</td><td>${name}</td><td>${e.wins}</td><td>${e.losses}</td><td>${e.draws}</td><td>${e.pointsFor}</td><td>${e.gamesPlayed}</td></tr>`;
  }).join('');

  content.innerHTML = `<table><thead><tr>${header}</tr></thead><tbody>${rows}</tbody></table>`;
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
