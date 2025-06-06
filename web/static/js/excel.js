document.addEventListener('DOMContentLoaded', function () {
    const alertModal = document.getElementById('alertModal');
    const modalTitle = document.getElementById('modalTitle');
    const modalMessage = document.getElementById('modalMessage');
    const modalIconContainer = document.getElementById('modalIconContainer');
    const closeModalButton = document.getElementById('closeModalButton');
    const uploadForm = document.getElementById('uploadForm');
    const uploadResultDiv = document.getElementById('uploadResult');
    const loadingIndicator = document.getElementById('loading'); // Pastikan ID ini ada di HTML

    const successIconSVG = `<svg class="h-6 w-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>`;
    const errorIconSVG = `<svg class="h-6 w-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`;

    let fullPreviewData = [];
    let currentPage = 1;
    const itemsPerPage = 50;
    let currentSessionId = null;
    let currentBrandNameValue = null;

    function showLoading(show) {
        if (loadingIndicator) {
            loadingIndicator.style.display = show ? 'flex' : 'none';
        }
    }

    function showModal(title, message, isSuccess) {
        if (!alertModal || !modalTitle || !modalMessage || !modalIconContainer) return;
        modalTitle.textContent = title;
        modalMessage.textContent = message;
        if (isSuccess) {
            modalIconContainer.innerHTML = successIconSVG;
            modalIconContainer.className = 'mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100';
        } else {
            modalIconContainer.innerHTML = errorIconSVG;
            modalIconContainer.className = 'mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100';
        }
        alertModal.classList.remove('hidden');
    }

    if (closeModalButton) {
        closeModalButton.addEventListener('click', function () {
            if(alertModal) alertModal.classList.add('hidden');
        });
    }
    
    function renderPreviewTable(page) {
        currentPage = page;
        const startIndex = (currentPage - 1) * itemsPerPage;
        const endIndex = startIndex + itemsPerPage;
        const paginatedData = fullPreviewData.slice(startIndex, endIndex);

        let tableHTML = '<div class="overflow-x-auto"><table class="min-w-full divide-y divide-gray-200">';
        tableHTML += '<thead class="bg-gray-50"><tr>';
        const headers = ["No", "Brand", "Username", "IP", "Last Login", "Membership", "Phone", "Email (Membership)", "Bank", "Account Name", "Account No", "Saldo", "Turnover", "Win/Loss", "Points", "Join Date", "Referral", "Uplink", "Status"];
        headers.forEach(header => {
            tableHTML += `<th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">${header}</th>`;
        });
        tableHTML += '</tr></thead><tbody class="bg-white divide-y divide-gray-200">';

        paginatedData.forEach((member, index) => {
            tableHTML += '<tr>';
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${startIndex + index + 1}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.brand_name || currentBrandNameValue || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.username || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.ip_address || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.last_login || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.membership_status || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.phone_number || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.membership_email || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.bank_name || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.account_name || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.account_no || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.saldo || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.turnover || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.win_loss || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.points || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.join_date || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.referral || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.uplink || ''}</td>`;
            tableHTML += `<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${member.status || ''}</td>`;
            tableHTML += '</tr>';
        });
        tableHTML += '</tbody></table></div>';
        
        const tableContainer = document.getElementById('previewTableContainer');
        if (tableContainer) {
            tableContainer.innerHTML = tableHTML;
        }
        renderPaginationControls();
    }

    function renderPaginationControls() {
        const paginationControlsDiv = document.getElementById('paginationControls');
        if (!paginationControlsDiv || fullPreviewData.length === 0) {
            if(paginationControlsDiv) paginationControlsDiv.innerHTML = '';
            return;
        }

        const totalPages = Math.ceil(fullPreviewData.length / itemsPerPage);
        let paginationHTML = '<nav aria-label="Page navigation"><ul class="inline-flex items-center -space-x-px">';

        // Previous Button
        paginationHTML += `<li><button data-page="${currentPage - 1}" class="page-link py-2 px-3 ml-0 leading-tight text-gray-500 bg-white border border-gray-300 rounded-l-lg hover:bg-gray-100 hover:text-gray-700 ${currentPage === 1 ? 'opacity-50 cursor-not-allowed' : ''}">Sebelumnya</button></li>`;

        // Page Numbers
        for (let i = 1; i <= totalPages; i++) {
            paginationHTML += `<li><button data-page="${i}" class="page-link py-2 px-3 leading-tight border border-gray-300 ${i === currentPage ? 'text-blue-600 bg-blue-50 hover:bg-blue-100 hover:text-blue-700' : 'text-gray-500 bg-white hover:bg-gray-100 hover:text-gray-700'}">${i}</button></li>`;
        }

        // Next Button
        paginationHTML += `<li><button data-page="${currentPage + 1}" class="page-link py-2 px-3 leading-tight text-gray-500 bg-white border border-gray-300 rounded-r-lg hover:bg-gray-100 hover:text-gray-700 ${currentPage === totalPages ? 'opacity-50 cursor-not-allowed' : ''}">Berikutnya</button></li>`;
        paginationHTML += '</ul></nav>';
        paginationControlsDiv.innerHTML = paginationHTML;

        // Add event listeners to page links
        paginationControlsDiv.querySelectorAll('.page-link').forEach(button => {
            button.addEventListener('click', function() {
                const page = parseInt(this.dataset.page);
                if (page) {
                    renderPreviewTable(page);
                }
            });
        });
    }

    async function handleConfirmImport() {
        showLoading(true);
        const importFormData = new FormData();

        // Debugging: Log nilai sebelum dikirim
        console.log("Nilai yang akan dikirim untuk impor:");
        console.log("currentSessionId:", currentSessionId);
        console.log("currentBrandNameValue:", currentBrandNameValue);

        importFormData.append('sessionId', currentSessionId);
        importFormData.append('brandName', currentBrandNameValue);

        try {
            const importResponse = await fetch('/upload/excel/import', {
                method: 'POST',
                body: importFormData,
            });
            const importResult = await importResponse.json();
            showLoading(false);

            if (importResult.success) {
                showModal("Impor Sukses", importResult.message, true);
                uploadResultDiv.innerHTML = ''; // Clear everything after successful import
                fullPreviewData = []; // Clear data
                currentSessionId = null;
                currentBrandNameValue = null;
                // Refresh halaman setelah beberapa saat agar pengguna bisa membaca modal
                setTimeout(() => {
                    window.location.reload();
                }, 2000); // Refresh setelah 2 detik
            } else {
                showModal("Impor Gagal", importResult.message, false);
            }
        } catch (err) {
            showLoading(false);
            showModal("Error", "Terjadi kesalahan saat impor: " + err.message, false);
        }
    }

    if (uploadForm) {
        uploadForm.addEventListener('submit', async function (event) {
            event.preventDefault();
            showLoading(true);
            uploadResultDiv.innerHTML = ''; // Clear previous preview entirely
            fullPreviewData = [];
            currentPage = 1;

            const formData = new FormData(uploadForm);
            currentBrandNameValue = formData.get("brandName"); 

            try {
                const response = await fetch('/upload/excel', {
                    method: 'POST',
                    body: formData,
                });

                const result = await response.json();
                showLoading(false);

                if (result.success && result.data) {
                    fullPreviewData = result.data;
                    currentSessionId = result.sessionId;
                    // currentBrandNameValue is already set from formData

                    // Create structure for controls and table
                    let previewStructureHTML = '<div class="bg-white rounded-xl shadow-sm p-6">';
                    previewStructureHTML += '<div class="mb-4 flex justify-between items-center">';
                    previewStructureHTML += `<div><span class="text-gray-700"><span class="font-medium">${fullPreviewData.length}</span> baris data ditemukan.</span></div>`;
                    previewStructureHTML += `
                        <button id="confirmImportBtnMain"
                                class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2">
                            <i class="fas fa-check mr-2"></i>Konfirmasi Import
                        </button>
                    `;
                    previewStructureHTML += '</div>';
                    previewStructureHTML += '<div id="paginationControls" class="mb-4"></div>'; 
                    previewStructureHTML += '<h2 class="text-xl font-bold mb-4">Preview Data</h2>';
                    previewStructureHTML += '<div id="previewTableContainer"></div>'; // Container for the table itself
                    previewStructureHTML += '</div>'; // Close bg-white wrapper

                    uploadResultDiv.innerHTML = previewStructureHTML;

                    // Add event listener for the main confirm button
                    const confirmBtnMain = document.getElementById('confirmImportBtnMain');
                    if(confirmBtnMain) {
                        confirmBtnMain.addEventListener('click', handleConfirmImport);
                    }

                    renderPreviewTable(1); // Render first page
                    // renderPaginationControls() is called by renderPreviewTable

                } else {
                    showModal("Kesalahan Pratinjau File", result.message || "Format respons tidak dikenal.", false);
                    uploadResultDiv.innerHTML = '';
                }
            } catch (err) {
                showLoading(false);
                showModal("Kesalahan Pratinjau File", "Terjadi kesalahan: " + err.message, false);
                uploadResultDiv.innerHTML = '';
            }
        });
    }
});

// Fungsi ini mungkin masih relevan jika input file dipertahankan dengan cara yang sama
function updateFileName(input) {
    const fileName = input.files[0] ? input.files[0].name : '';
    const selectedFileNameP = document.getElementById('selectedFileName');
    if (selectedFileNameP) {
        selectedFileNameP.textContent = fileName ? `File dipilih: ${fileName}` : '';
    }
}



