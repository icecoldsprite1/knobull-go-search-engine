document.addEventListener('DOMContentLoaded', () => {
    const searchBtn = document.getElementById('searchBtn');
    const goalInput = document.getElementById('goalInput');
    const categoryFilter = document.getElementById('categoryFilter');
    const typeFilter = document.getElementById('typeFilter');
    const resultsContainer = document.getElementById('resultsContainer');
    const loadingIndicator = document.getElementById('loadingIndicator');

    searchBtn.addEventListener('click', performSearch);
    goalInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });

    async function performSearch() {
        const goal = goalInput.value.trim();
        const category = categoryFilter.value;
        const type = typeFilter.value;
        if (!goal) return;

        resultsContainer.replaceChildren(); // Safer alternative to innerHTML = ''
        loadingIndicator.style.display = 'block';
        searchBtn.disabled = true;

        try {
            const response = await fetch('/api/recommend', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ goal, category, type })
            });

            if (!response.ok) throw new Error('Failed to fetch recommendations');

            const data = await response.json();
            renderResults(data);
        } catch (error) {
            console.error(error);
            resultsContainer.replaceChildren();
            const errorMsg = document.createElement('p');
            errorMsg.style.color = 'red';
            errorMsg.style.textAlign = 'center';
            errorMsg.textContent = 'Error fetching results.';
            resultsContainer.appendChild(errorMsg);
        } finally {
            loadingIndicator.style.display = 'none';
            searchBtn.disabled = false;
        }
    }

    function renderResults(results) {
        if (!results || results.length === 0) {
            resultsContainer.replaceChildren();
            const noResultsMsg = document.createElement('p');
            noResultsMsg.textContent = 'No matching resources found.';
            resultsContainer.appendChild(noResultsMsg);
            return;
        }

        results.forEach((resource) => {
            const card = document.createElement('div');
            card.className = 'card';
            
            const categorySpan = document.createElement('span');
            categorySpan.className = 'category';
            categorySpan.textContent = resource.category || 'General';
            card.appendChild(categorySpan);

            const titleH3 = document.createElement('h3');
            titleH3.textContent = resource.title;
            card.appendChild(titleH3);

            const descP = document.createElement('p');
            descP.textContent = resource.description;
            card.appendChild(descP);

            const actionDiv = document.createElement('div');
            actionDiv.className = 'card-action';

            let readBtn = null;
            let contentDiv = null;

            if (resource.type === 'external_course' || resource.type === 'tool') {
                const link = document.createElement('a');
                link.href = resource.link;
                link.target = '_blank';
                link.className = 'btn';
                link.textContent = 'View Link (External)';
                actionDiv.appendChild(link);
            } else if (resource.type === 'internal_article') {
                readBtn = document.createElement('button');
                readBtn.className = 'btn read-more-btn';
                readBtn.textContent = 'Read Article';
                actionDiv.appendChild(readBtn);

                contentDiv = document.createElement('div');
                contentDiv.className = 'article-content';
                contentDiv.textContent = resource.content;
            } else {
                // Fallback
                if (resource.link) {
                    const link = document.createElement('a');
                    link.href = resource.link;
                    link.target = '_blank';
                    link.className = 'btn';
                    link.textContent = 'View Resource';
                    actionDiv.appendChild(link);
                }
            }
            
            card.appendChild(actionDiv);
            if (contentDiv) {
                card.appendChild(contentDiv);
            }

            // Add event listener for reading internal articles
            if (readBtn && contentDiv) {
                readBtn.addEventListener('click', () => {
                    contentDiv.classList.toggle('show');
                    readBtn.textContent = contentDiv.classList.contains('show') ? 'Close Article' : 'Read Article';
                });
            }

            resultsContainer.appendChild(card);
        });
    }
});
