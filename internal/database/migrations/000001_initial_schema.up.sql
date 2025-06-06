-- Tabel Users
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE,
    role VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- Tabel Brands
CREATE TABLE IF NOT EXISTS brands (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    logo_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Members
CREATE TABLE IF NOT EXISTS members (
    id SERIAL PRIMARY KEY,
    username TEXT,
    ip_address TEXT,
    last_login TEXT,
    email TEXT,
    membership_status TEXT,
    phone_number TEXT,
    membership_email TEXT,
    bank_name TEXT,
    account_name TEXT,
    account_no TEXT,
    saldo TEXT,
    turnover TEXT,
    win_loss TEXT,
    points TEXT,
    join_date TEXT,
    referral TEXT,
    uplink TEXT,
    status TEXT,
    brand_name TEXT,
    crm_info TEXT,
    crm_user_id INTEGER,
    brand_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_members_crm_user_id FOREIGN KEY (crm_user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_members_brand_id FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_members_crm_user_id ON members(crm_user_id);
CREATE INDEX IF NOT EXISTS idx_members_brand_id ON members(brand_id);


-- Data Awal
INSERT INTO users (username, password, name, email, role) VALUES 
    ('admin', '$2a$10$W1MwBwX4dZjDAMPs7lzwUOhKH5NnrY7Iqa7lYpGWZdnGQ7iA5NznG', 'Administrator', 'admin@followup.id', 'admin'),
    ('superadmin', '$2a$10$BAPbAMhGgiGvN7tExb0LBOpgl3HG4vm0xPsFtc.gFOSe.lhbMl7t', 'Super Admin', 'tripraptomo24@gmail.com', 'superadmin')
ON CONFLICT (username) DO NOTHING;

-- Tabel Teams
CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    admin_user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_teams_name UNIQUE (name),
    CONSTRAINT uq_teams_admin_user_id UNIQUE (admin_user_id),
    CONSTRAINT fk_teams_admin_user FOREIGN KEY (admin_user_id) REFERENCES users(id) ON DELETE RESTRICT -- Admin tidak boleh dihapus jika masih jadi admin tim
);
CREATE INDEX IF NOT EXISTS idx_teams_admin_user_id ON teams(admin_user_id);

-- Tabel Team Members
CREATE TABLE IF NOT EXISTS team_members (
    id SERIAL PRIMARY KEY,
    team_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_team_members_user_id UNIQUE (user_id) -- Satu user hanya bisa di satu tim
);
CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON team_members(team_id);
-- Indeks pada user_id sudah tercakup oleh constraint UNIQUE uq_team_members_user_id 

-- Tabel Deposits
CREATE TABLE IF NOT EXISTS deposits (
    id SERIAL PRIMARY KEY,
    member_id INTEGER NOT NULL,
    amount NUMERIC(15, 2) NOT NULL, -- Menyimpan jumlah deposit, presisi 15 digit dengan 2 angka di belakang koma
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- Contoh status: pending, approved, rejected
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by_user_id INTEGER, -- User (CRM/Telemarketing) yang membuat request deposit
    approved_by_user_id INTEGER, -- User (Admin/Superadmin) yang menyetujui/menolak deposit
    approved_at TIMESTAMP, -- Waktu ketika deposit disetujui/dinyatakan
    rejection_reason TEXT, -- Alasan jika deposit ditolak
    CONSTRAINT fk_deposits_member FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE, -- Jika member dihapus, deposit terkait juga dihapus
    CONSTRAINT fk_deposits_created_by FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_deposits_approved_by FOREIGN KEY (approved_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_deposit_amount CHECK (amount > 0) -- Memastikan jumlah deposit positif
);

CREATE INDEX IF NOT EXISTS idx_deposits_member_id ON deposits(member_id);
CREATE INDEX IF NOT EXISTS idx_deposits_status ON deposits(status);
CREATE INDEX IF NOT EXISTS idx_deposits_created_by_user_id ON deposits(created_by_user_id);

