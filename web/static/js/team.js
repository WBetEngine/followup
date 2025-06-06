document.addEventListener('DOMContentLoaded', function () {
    const API_BASE_URL = '/api';
    const teamListContainer = document.getElementById('team-list-container');
    const teamTableBody = document.getElementById('team-table-body');
    const teamPaginationContainer = document.getElementById('team-pagination-container');
    const teamTablePlaceholder = document.getElementById('team-table-placeholder');

    const addTeamButton = document.getElementById('add-team-button');
    const teamModal = document.getElementById('team-modal');
    const teamModalContent = document.getElementById('team-modal-content');

    const manageMembersModal = document.getElementById('manage-members-modal');
    const manageMembersModalContent = document.getElementById('manage-members-modal-content');

    const searchTeamInput = document.getElementById('search-team');
    const applyTeamFilterButton = document.getElementById('apply-team-filter-button');

    let currentPage = 1;
    const defaultLimit = 10;
    let currentSearchTerm = '';

    // --- Utility Functions (Mirip user.js) ---
    function showLoadingIndicator(show) {
        const indicator = document.getElementById('loading-indicator');
        if (indicator) {
            indicator.style.display = show ? 'block' : 'none';
        }
    }

    function showToast(message, level = 'success') {
        // Implementasi toast notification sederhana (bisa diganti dengan library)
        const toastId = 'toast-' + Date.now();
        const toastElement = document.createElement('div');
        toastElement.id = toastId;
        toastElement.className = `fixed top-5 right-5 p-4 rounded-md shadow-lg text-white ${level === 'success' ? 'bg-green-500' : 'bg-red-500'}`;
        toastElement.textContent = message;
        document.body.appendChild(toastElement);
        setTimeout(() => {
            document.getElementById(toastId)?.remove();
        }, 3000);
    }

    function openModal(modalElement) {
        if (modalElement) {
            modalElement.classList.remove('hidden');
        }
    }

    function closeModal(modalElement, contentElement, loadingText) {
        if (modalElement) {
            modalElement.classList.add('hidden');
        }
        if (contentElement) {
            contentElement.innerHTML = `<div class="p-4 text-center"><i class="fas fa-spinner fa-spin text-2xl text-blue-600"></i><p class="mt-2 text-sm text-gray-500">${loadingText}</p></div>`;
        }
    }

    window.closeTeamModal = () => closeModal(teamModal, teamModalContent, 'Memuat form tim...');
    window.closeManageMembersModal = () => closeModal(manageMembersModal, manageMembersModalContent, 'Memuat detail anggota...');

    // --- Team Specific Functions ---

    async function fetchAndRenderTeams(page = 1, limit = defaultLimit, searchTerm = '') {
        showLoadingIndicator(true);
        if (teamTablePlaceholder) teamTablePlaceholder.innerHTML = 'Memuat data tim... <i class="fas fa-spinner fa-spin ml-2"></i>';
        
        try {
            const response = await fetch(`${API_BASE_URL}/teams?search=${encodeURIComponent(searchTerm)}&page=${page}&limit=${limit}`);
            if (!response.ok) {
                throw new Error(`Gagal memuat tim: ${response.statusText}`);
            }
            const data = await response.json();
            renderTeamTable(data.teams || []);
            renderTeamPagination(data);
            currentPage = page;
            currentSearchTerm = searchTerm;
        } catch (error) {
            console.error('Error fetching teams:', error);
            if (teamTableBody) teamTableBody.innerHTML = `<tr><td colspan="7" class="px-6 py-4 text-center text-red-500">Gagal memuat data tim: ${error.message}</td></tr>`;
            showToast(error.message, 'error');
        } finally {
            showLoadingIndicator(false);
            if (teamTablePlaceholder && teamTableBody.children.length > 1) teamTablePlaceholder.parentElement.remove(); // Hapus placeholder jika ada data
        }
    }

    function renderTeamTable(teams) {
        if (!teamTableBody) return;
        teamTableBody.innerHTML = ''; // Kosongkan tabel sebelum mengisi

        if (teams.length === 0) {
            teamTableBody.innerHTML = `<tr><td colspan="7" class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">Tidak ada tim ditemukan.</td></tr>`;
            return;
        }

        teams.forEach(team => {
            const desc = team.description && team.description.Valid ? team.description.String : '-';
            const row = `
                <tr data-team-id="${team.id}">
                    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">${team.id}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${escapeHTML(team.name)}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${escapeHTML(desc)}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${escapeHTML(team.admin_username || 'N/A')}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${team.member_count !== undefined ? team.member_count : 'N/A'}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${new Date(team.created_at).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour:'2-digit', minute:'2-digit' })}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <button type="button" class="text-blue-600 hover:text-blue-900 mr-2 manage-members-btn" data-team-id="${team.id}" data-team-name="${escapeHTML(team.name)}">Anggota</button>
                        <button type="button" class="text-indigo-600 hover:text-indigo-900 mr-2 edit-team-btn" data-team-id="${team.id}">Edit</button>
                        <button type="button" class="text-red-600 hover:text-red-900 delete-team-btn" data-team-id="${team.id}" data-team-name="${escapeHTML(team.name)}">Hapus</button>
                    </td>
                </tr>
            `;
            teamTableBody.insertAdjacentHTML('beforeend', row);
        });
        attachTeamActionListeners(); // Pasang listener setelah tabel di-render
    }

    function renderTeamPagination(data) {
        if (!teamPaginationContainer) return;
        teamPaginationContainer.innerHTML = ''; // Clear previous pagination

        const { total_records, current_page, per_page, total_pages } = data;

        if (!total_pages || total_pages <= 1) {
            teamPaginationContainer.innerHTML = '<p class="text-sm text-gray-700">Menampilkan semua hasil.</p>';
            return;
        }

        let paginationHTML = '<div class="flex-1 flex justify-between sm:hidden">';
        if (current_page > 1) {
            paginationHTML += `<button type="button" class="pagination-team-link relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50" data-page="${current_page - 1}">Sebelumnya</button>`;
        }
        if (current_page < total_pages) {
            paginationHTML += `<button type="button" class="pagination-team-link ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50" data-page="${current_page + 1}">Berikutnya</button>`;
        }
        paginationHTML += '</div>';

        paginationHTML += '<div class="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">';
        const startRecord = (current_page - 1) * per_page + 1;
        const endRecord = Math.min(current_page * per_page, total_records);
        paginationHTML += `<div><p class="text-sm text-gray-700">Menampilkan <span class="font-medium">${startRecord}</span> sampai <span class="font-medium">${endRecord}</span> dari <span class="font-medium">${total_records}</span> hasil</p></div>`;

        paginationHTML += '<div><nav class="relative z-0 inline-flex rounded-md shadow-sm -space-x-px" aria-label="Pagination">';
        
        // Previous button
        if (current_page > 1) {
             paginationHTML += `<button type="button" class="pagination-team-link relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50" data-page="${current_page - 1}"><span class="sr-only">Sebelumnya</span><i class="fas fa-chevron-left h-5 w-5"></i></button>`;
        } else {
             paginationHTML += `<span class="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-gray-100 text-sm font-medium text-gray-400 cursor-not-allowed"><span class="sr-only">Sebelumnya</span><i class="fas fa-chevron-left h-5 w-5"></i></span>`;
        }

        // Page links (simple version for now, can be enhanced like user.js)
        const maxPagesToShow = 5;
        let startPage = Math.max(1, current_page - Math.floor(maxPagesToShow / 2));
        let endPage = Math.min(total_pages, startPage + maxPagesToShow - 1);
        if (endPage - startPage + 1 < maxPagesToShow) {
            startPage = Math.max(1, endPage - maxPagesToShow + 1);
        }

        if (startPage > 1) {
            paginationHTML += `<button type="button" class="pagination-team-link bg-white border-gray-300 text-gray-500 hover:bg-gray-50 relative inline-flex items-center px-4 py-2 border text-sm font-medium" data-page="1">1</button>`;
            if (startPage > 2) {
                paginationHTML += `<span class="relative inline-flex items-center px-4 py-2 border border-gray-300 bg-white text-sm font-medium text-gray-700">...</span>`;
            }
        }

        for (let i = startPage; i <= endPage; i++) {
            paginationHTML += `<button type="button" class="pagination-team-link ${i === current_page ? 'z-10 bg-blue-50 border-blue-500 text-blue-600' : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'} relative inline-flex items-center px-4 py-2 border text-sm font-medium" data-page="${i}">${i}</button>`;
        }

        if (endPage < total_pages) {
            if (endPage < total_pages - 1) {
                paginationHTML += `<span class="relative inline-flex items-center px-4 py-2 border border-gray-300 bg-white text-sm font-medium text-gray-700">...</span>`;
            }
            paginationHTML += `<button type="button" class="pagination-team-link bg-white border-gray-300 text-gray-500 hover:bg-gray-50 relative inline-flex items-center px-4 py-2 border text-sm font-medium" data-page="${total_pages}">${total_pages}</button>`;
        }
        
        // Next button
        if (current_page < total_pages) {
            paginationHTML += `<button type="button" class="pagination-team-link relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50" data-page="${current_page + 1}"><span class="sr-only">Berikutnya</span><i class="fas fa-chevron-right h-5 w-5"></i></button>`;
        } else {
            paginationHTML += `<span class="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-gray-100 text-sm font-medium text-gray-400 cursor-not-allowed"><span class="sr-only">Berikutnya</span><i class="fas fa-chevron-right h-5 w-5"></i></span>`;
        }
        paginationHTML += '</nav></div></div>';
        teamPaginationContainer.innerHTML = paginationHTML;

        document.querySelectorAll('.pagination-team-link').forEach(button => {
            button.addEventListener('click', function() {
                const page = parseInt(this.dataset.page);
                fetchAndRenderTeams(page, defaultLimit, currentSearchTerm);
            });
        });
    }
    
    function escapeHTML(str) {
        if (str === null || str === undefined) return '';
        return str.toString().replace(/[&<>'"/]/g, function (s) {
            return {
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#39;',
                '/': '&#x2F;'
            }[s];
        });
    }

    async function loadTeamForm(teamId = null) {
        showLoadingIndicator(true);
        openModal(teamModal);
        teamModalContent.innerHTML = '<div class="p-4 text-center"><i class="fas fa-spinner fa-spin text-2xl text-blue-600"></i><p class="mt-2 text-sm text-gray-500">Memuat form tim...</p></div>';

        const isEditMode = teamId !== null;
        let formHtml = `
            <form id="team-action-form">
                <input type="hidden" id="form-team-id" value="${isEditMode ? teamId : ''}">
                <input type="hidden" id="form-is-edit-mode" value="${isEditMode ? 'true' : 'false'}">
                <div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                    <div class="sm:flex sm:items-start">
                        <div class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full ${isEditMode ? 'bg-yellow-100' : 'bg-blue-100'} sm:mx-0 sm:h-10 sm:w-10">
                            <i class="fas ${isEditMode ? 'fa-edit text-yellow-600' : 'fa-plus-circle text-blue-600'}"></i>
                        </div>
                        <div class="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left w-full">
                            <h3 class="text-lg leading-6 font-medium text-gray-900" id="team-modal-title">
                                ${isEditMode ? 'Edit Tim' : 'Tambah Tim Baru'}
                            </h3>
                            <div class="mt-4 space-y-4">
                                <div id="team-form-general-error" class="rounded-md bg-red-50 p-4" style="display:none;">
                                    <div class="flex">
                                        <div class="flex-shrink-0"><i class="fas fa-times-circle text-red-400"></i></div>
                                        <div class="ml-3"><h3 class="text-sm font-medium text-red-800">Error:</h3><div class="mt-2 text-sm text-red-700"><p id="team-form-general-error-text"></p></div></div>
                                    </div>
                                </div>
                                <div>
                                    <label for="team_name" class="block text-sm font-medium text-gray-700">Nama Tim <span class="text-red-500">*</span></label>
                                    <input type="text" name="team_name" id="team_name" required class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm">
                                    <p class="mt-1 text-xs text-red-500 field-error-text" id="team_name-error-text" style="display:none;"></p>
                                </div>
                                <div>
                                    <label for="team_description" class="block text-sm font-medium text-gray-700">Deskripsi</label>
                                    <textarea name="team_description" id="team_description" rows="3" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"></textarea>
                                    <p class="mt-1 text-xs text-red-500 field-error-text" id="team_description-error-text" style="display:none;"></p>
                                </div>
                                <div>
                                    <label for="admin_user_id" class="block text-sm font-medium text-gray-700">Admin Tim <span class="text-red-500">*</span></label>
                                    <select name="admin_user_id" id="admin_user_id" required class="mt-1 block w-full px-3 py-2 border border-gray-300 bg-white rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm">
                                        <option value="">Memuat admin...</option>
                                    </select>
                                    <p class="mt-1 text-xs text-red-500 field-error-text" id="admin_user_id-error-text" style="display:none;"></p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                    <button type="button" id="save-team-button" class="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 text-base font-medium text-white focus:outline-none focus:ring-2 focus:ring-offset-2 sm:ml-3 sm:w-auto sm:text-sm ${isEditMode ? 'bg-yellow-600 hover:bg-yellow-700 focus:ring-yellow-500' : 'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500'}">
                        ${isEditMode ? 'Simpan Perubahan' : 'Simpan Tim'}
                    </button>
                    <button type="button" onclick="closeTeamModal()" class="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm">
                        Batal
                    </button>
                </div>
            </form>
        `;
        teamModalContent.innerHTML = formHtml;
        
        const adminSelect = document.getElementById('admin_user_id');
        let currentAdminIdToExclude = isEditMode ? null : 0; // Untuk GetAssignableAdminsForTeamHandler

        // Populate admin dropdown
        try {
            if(isEditMode && teamId){
                const teamDetailResponse = await fetch(`${API_BASE_URL}/teams/${teamId}`);
                if (!teamDetailResponse.ok) throw new Error('Gagal memuat detail tim untuk form edit.');
                const teamData = await teamDetailResponse.json();
                document.getElementById('team_name').value = teamData.name;
                document.getElementById('team_description').value = teamData.description.Valid ? teamData.description.String : '';
                currentAdminIdToExclude = teamData.admin_user_id; // Exclude current admin from list, unless it's this team's admin
                await populateAdminDropdown(adminSelect, teamData.admin_user_id);
            } else {
                await populateAdminDropdown(adminSelect, null);
            }

        } catch (error) {
            console.error('Error populating admin dropdown:', error);
            adminSelect.innerHTML = '<option value="">Gagal memuat admin</option>';
            showToast('Gagal memuat daftar admin untuk form.', 'error');
        }

        attachTeamFormEventListeners();
        showLoadingIndicator(false);
    }
    
    async function populateAdminDropdown(selectElement, selectedAdminId) {
        if (!selectElement) return;
        try {
            // Parameter current_admin_id di API GetAssignableAdminsForTeamHandler akan meng-exclude ID tersebut jika disertakan
            // Untuk create, kita tidak exclude siapa-siapa. Untuk edit, kita akan exclude admin tim lain tapi BUKAN admin tim ini sendiri.
            // Logika exclude yang lebih baik mungkin perlu di sisi backend service.
            // Untuk sementara, saat edit, kita load semua assignable admin lalu pilih yang sekarang.
            const response = await fetch(`${API_BASE_URL}/users/assignable-as-admin?limit=100`); // Ambil semua
            if (!response.ok) throw new Error('Gagal mengambil daftar calon admin.');
            const admins = await response.json();
            
            selectElement.innerHTML = '<option value="">Pilih Admin</option>'; // Reset
            if (admins && admins.length > 0) {
                admins.forEach(admin => {
                    const option = document.createElement('option');
                    option.value = admin.id;
                    option.textContent = `${admin.name} (${admin.username})`;
                    if (selectedAdminId && admin.id === selectedAdminId) {
                        option.selected = true;
                    }
                    selectElement.appendChild(option);
                });
            } else {
                selectElement.innerHTML = '<option value="">Tidak ada admin tersedia</option>';
            }
        } catch (error) {
            console.error('Error fetching assignable admins:', error);
            selectElement.innerHTML = '<option value="">Error memuat admin</option>';
            showToast(error.message, 'error');
        }
    }

    function clearFormErrors() {
        document.querySelectorAll('.field-error-text').forEach(el => {
            el.textContent = '';
            el.style.display = 'none';
        });
        const generalError = document.getElementById('team-form-general-error');
        if (generalError) generalError.style.display = 'none';
    }

    function displayFormErrors(errors, generalErrorText) {
        clearFormErrors();
        if (errors) {
            for (const field in errors) {
                const errorElement = document.getElementById(`${field}-error-text`);
                if (errorElement) {
                    errorElement.textContent = errors[field];
                    errorElement.style.display = 'block';
                }
            }
        }
        if (generalErrorText) {
            const generalError = document.getElementById('team-form-general-error');
            const generalErrorMsg = document.getElementById('team-form-general-error-text');
            if (generalError && generalErrorMsg) {
                generalErrorMsg.textContent = generalErrorText;
                generalError.style.display = 'block';
            }
        }
    }

    async function handleSaveTeamForm() {
        clearFormErrors();
        showLoadingIndicator(true);

        const form = document.getElementById('team-action-form');
        const teamId = document.getElementById('form-team-id').value;
        const isEditMode = document.getElementById('form-is-edit-mode').value === 'true';

        const name = document.getElementById('team_name').value.trim();
        const descriptionInput = document.getElementById('team_description');
        const description = descriptionInput.value.trim() === '' ? null : descriptionInput.value.trim();
        const adminUserId = document.getElementById('admin_user_id').value;

        const payload = {
            name: name,
            description: description, // Ini akan jadi string atau null
            admin_user_id: parseInt(adminUserId)
        };
        
        // Untuk edit, field yang tidak diubah harusnya tidak dikirim atau dikirim sebagai null/pointer
        // API kita mengharapkan pointer untuk field opsional pada update.
        // Jika tidak ada perubahan, API service akan mengembalikan data yang ada.
        // JS akan selalu mengirim field, jadi backend harus bisa menangani ini.
        // Atau, kita bisa buat payload berbeda untuk create dan update.
        // Untuk sekarang, backend UpdateTeam sudah handle pointer, jadi ini seharusnya oke.

        const url = isEditMode ? `${API_BASE_URL}/teams/${teamId}` : `${API_BASE_URL}/teams`;
        const method = isEditMode ? 'PUT' : 'POST';

        try {
            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                    // 'X-CSRF-Token': '{{ .CSRFToken }}' // Jika CSRF token diperlukan
                },
                body: JSON.stringify(payload)
            });

            const responseData = await response.json();

            if (!response.ok) {
                let errorMessage = responseData.message || (isEditMode ? 'Gagal memperbarui tim.' : 'Gagal membuat tim.');
                if (response.status === 422 && responseData.errors) {
                    displayFormErrors(responseData.errors, "Data tidak valid.");
                } else {
                    displayFormErrors(null, errorMessage);
                }
                showToast(errorMessage, 'error');
                return; // Jangan tutup modal jika ada error
            }

            showToast(responseData.message || (isEditMode ? 'Tim berhasil diperbarui!' : 'Tim berhasil ditambahkan!'), 'success');
            closeTeamModal();
            fetchAndRenderTeams(currentPage, defaultLimit, currentSearchTerm); // Refresh list

        } catch (error) {
            console.error('Error saving team:', error);
            displayFormErrors(null, 'Terjadi kesalahan koneksi atau server.');
            showToast('Terjadi kesalahan: ' + error.message, 'error');
        } finally {
            showLoadingIndicator(false);
        }
    }

    async function handleDeleteTeam(teamId, teamName) {
        if (!confirm(`Apakah Anda yakin ingin menghapus tim "${teamName}"? Tindakan ini tidak dapat diurungkan.`)) {
            return;
        }
        showLoadingIndicator(true);
        try {
            const response = await fetch(`${API_BASE_URL}/teams/${teamId}`, {
                method: 'DELETE',
                headers: {
                    // 'X-CSRF-Token': '{{ .CSRFToken }}'
                }
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ message: 'Gagal menghapus tim.' }));
                throw new Error(errorData.message || `Gagal menghapus tim: ${response.statusText}`);
            }
            
            // Jika response.status === 200 atau 204, diasumsikan sukses
            let successMessage = `Tim "${teamName}" berhasil dihapus.`;
            if (response.status === 200) { // Jika API mengembalikan JSON message
                const responseData = await response.json();
                successMessage = responseData.message || successMessage;
            }

            showToast(successMessage, 'success');
            fetchAndRenderTeams(1, defaultLimit, ''); // Refresh ke halaman pertama
        } catch (error) {
            console.error('Error deleting team:', error);
            showToast('Gagal menghapus tim: ' + error.message, 'error');
        } finally {
            showLoadingIndicator(false);
        }
    }

    // --- Modal Kelola Anggota (Struktur Awal) ---
    async function loadManageMembersModal(teamId, teamName) {
        showLoadingIndicator(true);
        openModal(manageMembersModal);
        manageMembersModalContent.innerHTML = `<div class="p-4 text-center"><i class="fas fa-spinner fa-spin text-2xl text-blue-600"></i><p class="mt-2 text-sm text-gray-500">Memuat anggota tim ${escapeHTML(teamName)}...</p></div>`;

        try {
            let modalContentHtml = `
                <div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                    <div class="sm:flex sm:items-start">
                        <div class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-purple-100 sm:mx-0 sm:h-10 sm:w-10">
                            <i class="fas fa-users-cog text-purple-600"></i>
                        </div>
                        <div class="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left w-full">
                            <h3 class="text-lg leading-6 font-medium text-gray-900" id="manage-members-modal-title">
                                Kelola Anggota Tim: ${escapeHTML(teamName)}
                            </h3>
                            <input type="hidden" id="current-managing-team-id" value="${teamId}">
                            
                            <div class="mt-4" id="team-members-list-container">
                                <p class="text-sm text-gray-500">Memuat daftar anggota...</p>
                            </div>

                            <div class="mt-6 border-t pt-4">
                                <h4 class="text-md font-medium text-gray-800 mb-2">Tambah Anggota Baru</h4>
                                <div class="space-y-3">
                                    <div>
                                        <label for="search-new-member-input" class="sr-only">Cari Pengguna (CRM/Telemarketing)</label>
                                        <input type="text" id="search-new-member-input" placeholder="Ketik nama atau username..." class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-purple-500 focus:border-purple-500 sm:text-sm">
                                        <div id="assignable-users-results" class="mt-1 border border-gray-300 rounded-md shadow-sm max-h-40 overflow-y-auto" style="display:none;">
                                            <!-- Hasil pencarian user akan muncul di sini -->
                                        </div>
                                        <input type="hidden" id="selected-new-member-id">
                                        <p class="mt-1 text-xs text-red-500 field-error-text" id="add-member-error-text" style="display:none;"></p>

                                    </div>
                                    <button type="button" id="add-member-to-team-btn" class="inline-flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-purple-600 hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-purple-500 disabled:opacity-50" disabled>
                                        <i class="fas fa-plus mr-2"></i> Tambah ke Tim
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                    <button type="button" onclick="closeManageMembersModal()" class="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm">
                        Tutup
                    </button>
                </div>
            `;
            manageMembersModalContent.innerHTML = modalContentHtml;

            await fetchAndRenderTeamMembers(teamId);
            attachManageMembersListeners(teamId);

        } catch (error) {
            console.error('Error loading manage members modal:', error);
            showToast('Gagal memuat modal kelola anggota: ' + error.message, 'error');
        } finally {
            showLoadingIndicator(false);
        }
    }

    async function fetchAndRenderTeamMembers(teamId) {
        const membersListSection = document.getElementById('team-members-list-container');
        if (!membersListSection) return;
        membersListSection.innerHTML = '<p class="text-sm text-gray-500">Memuat anggota tim... <i class="fas fa-spinner fa-spin ml-1"></i></p>';
        
        try {
            const response = await fetch(`${API_BASE_URL}/teams/${teamId}/members`);
            if (!response.ok) throw new Error('Gagal memuat anggota tim.');
            const members = await response.json();
            renderTeamMembersList(members, teamId);
        } catch (error) {
            console.error(`Error fetching members for team ${teamId}:`, error);
            membersListSection.innerHTML = `<p class="text-sm text-red-500">Error: ${error.message}</p>`;
            showToast(error.message, 'error');
        }
    }

    function renderTeamMembersList(members, teamId) {
        const membersListSection = document.getElementById('team-members-list-container');
        if (!membersListSection) return;

        let membersHtml = '<h4 class="text-md font-medium text-gray-800 mb-2">Anggota Saat Ini:</h4>';
        if (!members || members.length === 0) {
            membersHtml += '<p class="text-sm text-gray-500">Belum ada anggota di tim ini.</p>';
        } else {
            membersHtml += '<ul class="divide-y divide-gray-200">';
            members.forEach(member => {
                // Cek apakah pengguna ini adalah admin tim, jika iya, jangan tampilkan tombol hapus
                // Ini memerlukan informasi admin tim, yang bisa didapat dari data tim yang di-load atau query tambahan
                // Untuk sementara, kita asumsikan kita tahu ID admin tim dari tempat lain jika perlu
                // Atau, kita bisa dapatkan dari `team.admin_user_id` saat `loadManageMembersModal` dipanggil
                // dan membandingkannya dengan `member.user_id`.
                // Untuk sekarang, kita sederhanakan dan tombol hapus selalu ada (akan dihandle backend)
                membersHtml += `
                    <li class="py-3 flex justify-between items-center">
                        <div>
                            <p class="text-sm font-medium text-gray-900">${escapeHTML(member.user_full_name)} (${escapeHTML(member.username)})</p>
                            <p class="text-xs text-gray-500">Peran: ${escapeHTML(member.user_role)} - Bergabung: ${new Date(member.joined_at).toLocaleDateString('id-ID')}</p>
                        </div>
                        <button type="button" class="remove-member-btn text-red-500 hover:text-red-700 text-sm" data-user-id="${member.user_id}" data-team-id="${teamId}" data-member-name="${escapeHTML(member.user_full_name)}">
                            <i class="fas fa-trash-alt mr-1"></i> Hapus
                        </button>
                    </li>
                `;
            });
            membersHtml += '</ul>';
        }
        membersListSection.innerHTML = membersHtml;
        attachRemoveMemberListeners(teamId);
    }

    function renderAssignableUsers(users, container, selectedInput, searchInput) {
        container.innerHTML = '';
        const addMemberButton = document.getElementById('add-member-to-team-btn');

        if (users && users.length > 0) {
            users.forEach(user => {
                const div = document.createElement('div');
                div.className = 'p-2 hover:bg-gray-100 cursor-pointer text-sm';
                div.textContent = `${user.name} (${user.username})`;
                div.dataset.userId = user.id;
                div.addEventListener('click', function() {
                    selectedInput.value = user.id;
                    searchInput.value = `${user.name} (${user.username})`; 
                    container.style.display = 'none'; 
                    if (addMemberButton) {
                        addMemberButton.disabled = false; // Aktifkan tombol
                    }
                });
                container.appendChild(div);
            });
        } else {
            container.innerHTML = '<div class="p-2 text-sm text-gray-500">Tidak ada pengguna ditemukan.</div>';
            if (addMemberButton) {
                addMemberButton.disabled = true; // Pastikan tombol disabled jika tidak ada hasil
            }
        }
    }

    function attachManageMembersListeners(teamId) {
        const searchInput = document.getElementById('search-new-member-input');
        const resultsContainer = document.getElementById('assignable-users-results');
        const selectedUserInput = document.getElementById('selected-new-member-id');
        const addMemberButton = document.getElementById('add-member-to-team-btn');
        
        let searchTimeout;
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                clearTimeout(searchTimeout);
                const searchTerm = this.value.trim();
                selectedUserInput.value = ''; // Kosongkan ID pengguna yang dipilih
                resultsContainer.innerHTML = ''; // Bersihkan hasil lama
                if (addMemberButton) {
                    addMemberButton.disabled = true; // Nonaktifkan tombol saat input berubah
                }
                if (searchTerm.length < 2) { // Panjang minimal untuk mulai mencari, bisa disesuaikan
                    resultsContainer.style.display = 'none';
                    return;
                }
                resultsContainer.style.display = 'block';
                resultsContainer.innerHTML = '<div class="p-2 text-sm text-gray-500">Mencari...</div>';
                searchTimeout = setTimeout(async () => {
                    try {
                        const response = await fetch(`${API_BASE_URL}/users/assignable-to-team?search=${encodeURIComponent(searchTerm)}&limit=10`);
                        if (!response.ok) throw new Error('Gagal mencari pengguna.');
                        const users = await response.json();
                        renderAssignableUsers(users, resultsContainer, selectedUserInput, searchInput);
                    } catch (error) {
                        resultsContainer.innerHTML = `<div class="p-2 text-sm text-red-500">Error: ${error.message}</div>`;
                    }
                }, 500);
            });
        }

        if (addMemberButton) {
            addMemberButton.addEventListener('click', async function() {
                const userIdToAdd = selectedUserInput.value;
                const errorDisplayElement = document.getElementById('add-member-error-text');

                if (errorDisplayElement) {
                    errorDisplayElement.textContent = '';
                    errorDisplayElement.style.display = 'none';
                }

                if (!userIdToAdd) {
                    if (errorDisplayElement) {
                        errorDisplayElement.textContent = 'Silakan pilih pengguna untuk ditambahkan.';
                        errorDisplayElement.style.display = 'block';
                    }
                    return;
                }

                showLoadingIndicator(true);
                try {
                    const response = await fetch(`${API_BASE_URL}/teams/${teamId}/members`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ user_id: parseInt(userIdToAdd) })
                    });
                    const responseData = await response.json();
                    if (!response.ok) {
                        let errMsg = responseData.message || 'Gagal menambahkan anggota.';
                        if (responseData.errors && responseData.errors.user_id) {
                            errMsg = responseData.errors.user_id;
                        }
                        if (errorDisplayElement) {
                            errorDisplayElement.textContent = errMsg;
                            errorDisplayElement.style.display = 'block';
                        }
                        throw new Error(errMsg); 
                    }
                    showToast(responseData.message || 'Anggota berhasil ditambahkan!', 'success');
                    fetchAndRenderTeamMembers(teamId); // Refresh list anggota
                    fetchAndRenderTeams(currentPage, defaultLimit, currentSearchTerm); // Refresh list tim (untuk member count)
                    if(searchInput) searchInput.value = ''; // Clear search
                    if(resultsContainer) {
                        resultsContainer.innerHTML = '';
                        resultsContainer.style.display = 'none';
                    }
                    if(selectedUserInput) selectedUserInput.value = '';
                    if(addMemberButton) addMemberButton.disabled = true; // Nonaktifkan tombol lagi setelah berhasil
                } catch (error) {
                    console.error('Error adding member:', error);
                    showToast(error.message, 'error');
                } finally {
                    showLoadingIndicator(false);
                }
            });
        }
    }

    function attachRemoveMemberListeners(teamId) {
        document.querySelectorAll('.remove-member-btn').forEach(button => {
            // Hapus event listener lama jika ada untuk menghindari multiple attachment
            const newButton = button.cloneNode(true);
            button.parentNode.replaceChild(newButton, button);

            newButton.addEventListener('click', async function() {
                const userId = this.dataset.userId;
                const memberName = this.dataset.memberName;
                if (!confirm(`Apakah Anda yakin ingin menghapus ${memberName} dari tim ini?`)) return;

                showLoadingIndicator(true);
                try {
                    const response = await fetch(`${API_BASE_URL}/teams/${teamId}/members/${userId}`, { method: 'DELETE' });
                    if (!response.ok) {
                        const errorData = await response.json().catch(() => ({}));
                        throw new Error(errorData.message || 'Gagal menghapus anggota.');
                    }
                    showToast('Anggota berhasil dihapus.', 'success');
                    fetchAndRenderTeamMembers(teamId);
                    fetchAndRenderTeams(currentPage, defaultLimit, currentSearchTerm); // Refresh list tim (untuk member count)
                } catch (error) {
                    console.error('Error removing member:', error);
                    showToast(error.message, 'error');
                } finally {
                    showLoadingIndicator(false);
                }
            });
        });
    }

    // --- Event Listeners --- 
    function attachTeamActionListeners() {
        // Tombol Edit Tim
        document.querySelectorAll('.edit-team-btn').forEach(button => {
            button.addEventListener('click', function() {
                const teamId = this.dataset.teamId;
                loadTeamForm(teamId);
            });
        });

        // Tombol Hapus Tim
        document.querySelectorAll('.delete-team-btn').forEach(button => {
            button.addEventListener('click', function() {
                const teamId = this.dataset.teamId;
                const teamName = this.dataset.teamName;
                handleDeleteTeam(teamId, teamName);
            });
        });

        // Tombol Kelola Anggota
        document.querySelectorAll('.manage-members-btn').forEach(button => {
            button.addEventListener('click', function() {
                const teamId = this.dataset.teamId;
                const teamName = this.dataset.teamName;
                loadManageMembersModal(teamId, teamName);
            });
        });
    }

    function attachTeamFormEventListeners() {
        const saveButton = document.getElementById('save-team-button');
        if (saveButton) {
            saveButton.addEventListener('click', handleSaveTeamForm);
        }
        // Event listener untuk input form bisa ditambahkan di sini jika perlu validasi real-time
    }

    if (addTeamButton) {
        addTeamButton.addEventListener('click', () => loadTeamForm());
    }

    if (applyTeamFilterButton && searchTeamInput) {
        applyTeamFilterButton.addEventListener('click', () => {
            fetchAndRenderTeams(1, defaultLimit, searchTeamInput.value.trim());
        });
        searchTeamInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                fetchAndRenderTeams(1, defaultLimit, searchTeamInput.value.trim());
            }
        });
    }

    // Inisialisasi: Muat daftar tim saat halaman pertama kali dibuka
    fetchAndRenderTeams();
});
