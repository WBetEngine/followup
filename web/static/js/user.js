// Pindahkan semua JavaScript dari user.html ke sini

document.addEventListener('DOMContentLoaded', function () {
    const userListContainer = document.getElementById('user-list-container');
    const modal = document.getElementById('user-modal');
    const modalContent = document.getElementById('modal-content');
    const addUserButton = document.getElementById('add-user-button');
    const filterForm = document.getElementById('user-filter-form');
    const searchInput = document.getElementById('search');
    const roleFilterInput = document.getElementById('role-filter');
    const loadingIndicator = document.getElementById('loading-indicator'); // Referensi ke loading indicator

    // Default values
    let currentPage = 1;
    const defaultLimit = 25; // Sesuaikan dengan backend atau buat konstan
    const SPINNER_HTML = '<div class="p-4 text-center"><i class="fas fa-spinner fa-spin text-2xl text-blue-600"></i><p class="mt-2 text-sm text-gray-500">Memuat...</p></div>';
    const SPINNER_EDIT_HTML = '<div class="p-4 text-center"><i class="fas fa-spinner fa-spin text-2xl text-blue-600"></i><p class="mt-2 text-sm text-gray-500">Memuat form edit...</p></div>';

    function showLoadingIndicator(show) {
        if (loadingIndicator) {
            loadingIndicator.style.display = show ? 'block' : 'none';
        }
    }

    async function fetchAndRenderUsers(page = 1, searchTerm = '', roleFilter = '') {
        currentPage = page; // Update current page
        const search = searchTerm || (searchInput ? searchInput.value : '');
        const role = roleFilter || (roleFilterInput ? roleFilterInput.value : '');
        
        showLoadingIndicator(true);
        try {
            const response = await fetch(`/user?page=${page}&limit=${defaultLimit}&search=${encodeURIComponent(search)}&role=${encodeURIComponent(role)}&fragment=true`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const html = await response.text();
            if (userListContainer) {
                userListContainer.innerHTML = html;
            }
            attachActionListeners(); // Pasang listener ke tombol-tombol baru di tabel
        } catch (error) {
            console.error('Error fetching users:', error);
            if (userListContainer) {
                userListContainer.innerHTML = '<p class="text-red-500 p-4">Gagal memuat data pengguna. Silakan coba lagi.</p>';
            }
        } finally {
            showLoadingIndicator(false);
        }
    }

    async function loadUserForm(userId = null) {
        const url = userId ? `/user/form/edit/${userId}` : '/user/form/add';
        console.log(`[loadUserForm] Fetching form from URL: ${url}`);
        // Spinner sudah ditampilkan oleh pemanggil (addUserButton listener atau edit-user-btn listener)
        // Modal juga sudah ditampilkan oleh pemanggil
        showLoadingIndicator(true); // Indikator global fetch (jika berbeda dari spinner modal)
        try {
            const response = await fetch(url);
            console.log(`[loadUserForm] Response status: ${response.status}, ok: ${response.ok}`);
            if (!response.ok) {
                const errorText = await response.text();
                console.error(`[loadUserForm] HTTP error! Status: ${response.status}, Text: ${errorText}`);
                throw new Error(`HTTP error! status: ${response.status} - ${response.statusText}. Detail: ${errorText}`);
            }
            const html = await response.text();
            console.log("[loadUserForm] Successfully fetched HTML, length:", html.length);
            modalContent.innerHTML = html;
            // modal.classList.remove('hidden'); // Sudah di-handle oleh pemanggil
            attachFormEventListeners();
            console.log("[loadUserForm] Form rendered and event listeners attached.");
        } catch (error) {
            console.error('[loadUserForm] Error loading user form:', error);
            modalContent.innerHTML = `<div class="p-4"><p class="text-red-600 font-semibold">Gagal memuat form.</p><p class="text-sm text-gray-700 mt-1">Detail: ${error.message}</p><p class="text-xs text-gray-500 mt-2">Silakan periksa konsol browser untuk log lebih lanjut dan coba lagi. Jika masalah berlanjut, hubungi administrator.</p></div>`;
            // modal.classList.remove('hidden'); // Pastikan modal tetap terlihat dengan pesan error
        } finally {
            showLoadingIndicator(false); // Indikator global fetch
            console.log("[loadUserForm] Finished execution.");
        }
    }

    function clearFormErrors() {
        const generalError = document.getElementById('user-form-general-error');
        if (generalError) generalError.style.display = 'none';
        
        const generalErrorText = document.getElementById('user-form-general-error-text');
        if (generalErrorText) generalErrorText.textContent = '';

        document.querySelectorAll('.field-error-text').forEach(el => {
            el.textContent = '';
            el.style.display = 'none';
        });
        document.querySelectorAll('#user-action-form input, #user-action-form select').forEach(el => {
            el.classList.remove('border-red-500');
        });
    }

    function displayFormErrors(errors, generalMessage = null) {
        clearFormErrors();
        if (generalMessage) {
            const generalError = document.getElementById('user-form-general-error');
            const generalErrorText = document.getElementById('user-form-general-error-text');
            if (generalErrorText) generalErrorText.textContent = generalMessage;
            if (generalError) generalError.style.display = 'block';
        }

        if (errors) {
            for (const field in errors) {
                const errorEl = document.getElementById(`${field}-error-text`);
                const inputEl = document.getElementById(field);
                if (errorEl) {
                    errorEl.textContent = errors[field];
                    errorEl.style.display = 'block';
                }
                if (inputEl) {
                    inputEl.classList.add('border-red-500');
                }
            }
        }
    }

    async function handleSaveUserForm(event) {
        event.preventDefault();
        clearFormErrors();
        showLoadingIndicator(true);

        const form = document.getElementById('user-action-form');
        const userId = document.getElementById('form-user-id').value;
        const isEditMode = document.getElementById('form-is-edit-mode').value === 'true';

        const formData = new FormData(form);
        // Untuk CreateUser (form-urlencoded), username ada di formData
        // Untuk UpdateUser (JSON), kita perlu mengambilnya secara manual jika ingin digunakan di pesan sukses
        const nameForMessage = formData.get('name') || (isEditMode ? 'User' : 'User baru'); 

        const payload = Object.fromEntries(formData.entries());
        
        if (isEditMode) {
            delete payload.username; // Username tidak boleh diubah dan tidak dikirim untuk update
        }

        const url = isEditMode ? `/user/update/${userId}` : '/user/create';
        const method = 'POST';

        try {
            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': isEditMode ? 'application/json' : 'application/x-www-form-urlencoded',
                    'Accept': 'application/json'
                },
                body: isEditMode ? JSON.stringify(payload) : new URLSearchParams(formData).toString()
            });

            if (response.ok) {
                // Selalu coba parse JSON jika response.ok, karena baik create maupun update sekarang mengembalikan JSON sukses
                const result = await response.json().catch(err => {
                    console.error("Error parsing success JSON:", err);
                    // Fallback jika parsing JSON sukses gagal (seharusnya tidak terjadi untuk 200/201 dari handler kita)
                    return { message: isEditMode ? "User berhasil diperbarui (respons tidak standar)." : "User berhasil ditambahkan (respons tidak standar)." };
                });
                
                alert(result.message || (isEditMode ? "User berhasil diperbarui!" : "User berhasil ditambahkan!"));
                modal.classList.add('hidden');
                modalContent.innerHTML = ''; // Kosongkan konten modal
                fetchAndRenderUsers(isEditMode ? currentPage : 1); // Refresh tabel, ke halaman 1 jika tambah user baru
            } else {
                const errorData = await response.json().catch(() => ({ 
                    message: `Terjadi kesalahan (${response.status}). Respons tidak valid.` 
                }));
                
                if (response.status === 422) { // Error validasi
                    if (errorData.errors) { // Struktur dari UpdateUserHandler
                        displayFormErrors(errorData.errors, 'Periksa kembali input Anda.');
                    } else if (errorData.Data && errorData.Data.ValidationErrors) { // Struktur dari CreateUserHandler (jika mengembalikan HTML error)
                        displayFormErrors(errorData.Data.ValidationErrors, errorData.Data.FormError || 'Periksa kembali input Anda.');
                    } else {
                         displayFormErrors(null, errorData.message || 'Error validasi tidak dikenal.');
                    }
                } else if (errorData.Data && errorData.Data.FormError) { // Error umum dari CreateUserHandler (HTML error)
                    displayFormErrors(null, errorData.Data.FormError);
                } else { // Error umum lainnya
                    displayFormErrors(null, errorData.message || errorData.error || `Gagal menyimpan pengguna (${response.status}).`);
                }
            }
        } catch (error) {
            console.error('Error saving user:', error);
            displayFormErrors(null, 'Terjadi kesalahan jaringan atau request gagal.');
        } finally {
            showLoadingIndicator(false);
        }
    }
    
    async function handleDeleteUser(userId, username) {
        if (confirm(`Apakah Anda yakin ingin menghapus user "${username}" (ID: ${userId})?`)) {
            showLoadingIndicator(true);
            try {
                const response = await fetch(`/user/delete/${userId}`, { method: 'POST' });
                if (response.ok && response.status === 204) { 
                    alert(`User '${username}' (ID: ${userId}) berhasil dihapus!`);
                    fetchAndRenderUsers(currentPage); 
                } else {
                    const errorData = await response.json().catch(() => ({ message: `Gagal menghapus user. Status: ${response.status}` }));
                    alert(errorData.message || `Gagal menghapus user ID ${userId}.`);
                }
            } catch (error) {
                console.error('Error deleting user:', error);
                alert('Terjadi kesalahan saat menghapus pengguna.');
            } finally {
                showLoadingIndicator(false);
            }
        }
    }

    function attachFormEventListeners() {
        const userActionForm = document.getElementById('user-action-form');
        if (userActionForm) {
            userActionForm.removeEventListener('submit', handleSaveUserForm); 
            userActionForm.addEventListener('submit', handleSaveUserForm); 
        }
        
        const saveButton = document.getElementById('save-user-button');
        if (saveButton) {
            saveButton.removeEventListener('click', handleSaveUserForm); 
            saveButton.addEventListener('click', handleSaveUserForm);
        }

        const cancelButton = document.getElementById('cancel-user-form-button');
        if (cancelButton) {
            cancelButton.onclick = () => { 
                modal.classList.add('hidden');
                modalContent.innerHTML = '';
            };
        }
    }
    
    function attachActionListeners() {
        userListContainer.querySelectorAll('.edit-user-btn').forEach(button => {
            button.addEventListener('click', function() {
                const userId = this.dataset.userId;
                modalContent.innerHTML = SPINNER_EDIT_HTML;
                modal.classList.remove('hidden');
                loadUserForm(userId);
            });
        });

        userListContainer.querySelectorAll('.delete-user-btn').forEach(button => {
            button.addEventListener('click', function() {
                const userId = this.dataset.userId;
                const username = this.dataset.username;
                handleDeleteUser(userId, username);
            });
        });
        
        userListContainer.querySelectorAll('.pagination-link').forEach(link => {
            link.addEventListener('click', function(event) {
                event.preventDefault();
                const page = this.dataset.page;
                if (page) {
                    fetchAndRenderUsers(parseInt(page));
                }
            });
        });
    }

    if (filterForm) {
        filterForm.addEventListener('submit', function (event) {
            event.preventDefault(); 
            fetchAndRenderUsers(1); 
        });
    }
    
    const applyFilterButton = document.getElementById('apply-filter-button');
    if (applyFilterButton && !filterForm.contains(applyFilterButton)) { 
         applyFilterButton.addEventListener('click', () => fetchAndRenderUsers(1));
    }

    if (addUserButton) {
        addUserButton.addEventListener('click', () => {
            modalContent.innerHTML = SPINNER_HTML;
            modal.classList.remove('hidden');
            loadUserForm();
        });
    }
    
    if (userListContainer && userListContainer.innerHTML.trim() === '') {
        fetchAndRenderUsers(currentPage);
    } else if (userListContainer) {
        attachActionListeners();
    }
});
