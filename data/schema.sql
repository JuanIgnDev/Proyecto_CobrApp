PRAGMA foreign_keys = ON;

-- Cliente 
CREATE TABLE IF NOT EXISTS cliente (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre      TEXT NOT NULL,
    apellido    TEXT NOT NULL,
    email       TEXT UNIQUE,
    telefono    TEXT,
    creado_en   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Venta
CREATE TABLE IF NOT EXISTS venta (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id   INTEGER NOT NULL,
    total        REAL NOT NULL CHECK (total >= 0),
    descripcion  TEXT,
    fecha        TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (cliente_id) REFERENCES cliente(id) ON DELETE CASCADE
);

-- Cobro
CREATE TABLE IF NOT EXISTS cobro (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id   INTEGER NOT NULL,
    monto        REAL NOT NULL CHECK (monto > 0),
    observacion  TEXT,
    fecha        TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (cliente_id) REFERENCES cliente(id) ON DELETE CASCADE
);

-- Notificacion
CREATE TABLE IF NOT EXISTS notificacion (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id INTEGER NOT NULL,
    tipo TEXT NOT NULL,
    titulo TEXT NOT NULL,
    fecha_referencia TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'pendiente',
    fecha_creacion TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (cliente_id) REFERENCES cliente(id) ON DELETE CASCADE
);

-- Aviso
CREATE TABLE IF NOT EXISTS aviso (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id   INTEGER NOT NULL,
    tipo         TEXT NOT NULL,
    fecha        TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (cliente_id) REFERENCES cliente(id) ON DELETE CASCADE
);

-- Índices 
CREATE INDEX IF NOT EXISTS idx_venta_cliente ON venta(cliente_id);
CREATE INDEX IF NOT EXISTS idx_cobro_cliente ON cobro(cliente_id);
CREATE INDEX IF NOT EXISTS idx_aviso_cliente ON aviso(cliente_id);

-- Vista de saldo
CREATE VIEW IF NOT EXISTS vista_saldo_cliente AS
SELECT
    c.id AS cliente_id,
    c.nombre,
    c.apellido,
    c.email,
    c.telefono,
    COALESCE((SELECT SUM(total) FROM venta WHERE venta.cliente_id = c.id), 0) AS total_vendido,
    COALESCE((SELECT SUM(monto) FROM cobro WHERE cobro.cliente_id = c.id), 0) AS total_cobrado,
    COALESCE((SELECT SUM(total) FROM venta WHERE venta.cliente_id = c.id), 0)
      - COALESCE((SELECT SUM(monto) FROM cobro WHERE cobro.cliente_id = c.id), 0) AS saldo_pendiente
FROM cliente c;

-- Configuracion
CREATE TABLE IF NOT EXISTS configuracion (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    dias_alerta INTEGER NOT NULL,
    mensaje_wp TEXT NOT NULL,
    minutos_inactividad INTEGER CHECK (minutos_inactividad >= 0)
);

INSERT OR IGNORE INTO configuracion (id,dias_alerta,mensaje_wp, minutos_inactividad)
VALUES (1, 45, 'Hola {nombre}, te escribo para recordarte que tenés un saldo pendiente de ${saldo}. Avisame cuando puedas regularizarlo. ¡Saludos!',30);