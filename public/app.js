document.addEventListener('DOMContentLoaded', () => {
    const searchBtn = document.getElementById('searchBtn');
    const goalInput = document.getElementById('goalInput');
    const resultsContainer = document.getElementById('resultsContainer');
    const loadingIndicator = document.getElementById('loadingIndicator');

    searchBtn.addEventListener('click', performSearch);
    goalInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });

    async function performSearch() {
        const goal = goalInput.value.trim();
        if (!goal) return;

        // UI state update
        resultsContainer.innerHTML = '';
        loadingIndicator.style.display = 'block';
        searchBtn.disabled = true;

        try {
            const response = await fetch('/api/recommend', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ goal })
            });

            if (!response.ok) throw new Error('Failed to fetch recommendations');

            const data = await response.json();
            renderResults(data);
        } catch (error) {
            console.error(error);
            resultsContainer.innerHTML = `<p style="color: red; text-align: center;">Error fetching results. Make sure your Go backend is running!</p>`;
        } finally {
            loadingIndicator.style.display = 'none';
            searchBtn.disabled = false;
        }
    }

    function renderResults(results) {
        if (!results || results.length === 0) {
            resultsContainer.innerHTML = `<p style="text-align: center; color: var(--text-muted); font-size: 1.1rem;">No matching resources found for that goal. Try something else!</p>`;
            return;
        }

        results.forEach((resource, index) => {
            const card = document.createElement('div');
            card.className = 'card';
            // Stagger animation delay slightly for each card
            card.style.animationDelay = `${index * 0.1}s`;
            
            card.innerHTML = `
                <span class="category">${resource.category || 'General'}</span>
                <h3>${resource.title}</h3>
                <p>${resource.description}</p>
            `;
            resultsContainer.appendChild(card);
        });
    }
});
