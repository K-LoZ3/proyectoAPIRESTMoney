package main

import (
  "net/http"
  "time"
  "fmt"
  "database/sql"
  "encoding/json"
  "log"
  
  _ "modernc.org/sqlite"
  "github.com/gorilla/mux"
)

//Eliminar al finalizar con los handlers
func holaMundo(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Hola Mundo!")
}

//Estructura movimiento: para la base de datos manejaremos los movimientos positivos y negativos
//con la misma estructura. El campo tipo sera el que ayude a identificar si el valor es un egreso
//o un ingreso y le dara la naturaleza al movimiento. En la bd se usara una tabla.
type Movimiento struct {
  Id int `json:"id"`
  Tipo string `json:"tipo"`
  Monto int `json:"monto"`
  Descripcion string `json:"descripcion"`
  Grupo string `json:"grupo"`
  Fecha time.Time `json:"fecha"`
  Creado time.Time `json:"creado"`
}

var db *sql.DB

func initDB() {
  var err error
  db, err = sql.Open("sqlite", "movimientos.db")
  if err != nil {
    log.Fatal(err)
  }
  
  crearTabla := `
  CREATE TABLE IF NOT EXISTS movimientos(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tipo TEXT,
  monto INTEGER,
  descripcion TEXT,
  grupo TEXT,
  fecha DATETIME,
  creado DATETIME
  );`
  
  _, err = db.Exec(crearTabla)
  if err != nil {
    log.Fatal("Error creando la tabla", err)
  }
}

//comprobarMovimiento recibe la estructura para validar si se ingrresaron los datos
//obligatorios. Devuelve un error si falta  datos.
func comprobarMovimiento(m Movimiento) error{
  if m.Monto == 0 || m.Fecha.IsZero() {
    return fmt.Errorf("Error al ingresar los datos, datos importantes son omitidos")
  }
  return nil
}

//getEgresos consulta los egresos en la tabla, que sean egresos y luego
//los envia en formati json al navegador.
func getEgresos(w http.ResponseWriter, r *http.Request) {
  //consultamos en la tabla los egresos
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos WHERE tipo = ?", "egreso")
  //Comprobamos el error
  if err != nil {
    http.Error(w, "Error al leer los datos de la tabla.", http.StatusBadRequest)
   return 
  }
  defer rows.Close() //cerramos la base de datos
  
  //Creamos el slite para agrupar todos los egresos
  var movimientos []Movimiento
  //recorremos cada fila co  un for.
  for rows.Next() {
    var m Movimiento
    //Escaneamos los datos por cada fila y los pasamos a la estructura
    //validamos el error al mismo tiempo
    if err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado) ; err != nil {
      http.Error(w, "Error al pasar los datos de la tabla a la estructura.", http.StatusBadRequest)
      return
    }
    //agregamls los datos a a la estructura
    movimientos = append(movimientos, m)
  }
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slite a json y los enviamos
  //al usuario, valodamos el error
  err = json.NewEncoder(w).Encode(movimientos)
  if err != nil {
    http.Error(w, "error al enviar los datos del getEgresos", http.StatusInternalServerError)
  }
}

func getIngresos(w http.ResponseWriter, r *http.Request) {
  //consultamos los movimientos tipo ingreso, validamos el error.
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos WHERE tipo = ?", "ingreso")
  if err != nil {
    http.Error(w, "Error al consultar los ingresos el la db.", http.StatusBadRequest)
    return
  }
  defer rows.Close()
  
  //Creo la variable para almacenar los movimientos
  var movimientos []Movimiento
  for rows.Next() {
    var m Movimiento
    //Escaneamos los datos de la cada fila
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado)
    if err != nil {
      http.Error(w, "Error al scanear los datos de la fila.", http.StatusBadRequest)
      return
    }
    //Agregamos cada fila/structura al slite 
    movimientos = append(movimientos, m)
  }
  
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slite a json y los enviamos
  //al usuario, valodamos el error
  err = json.NewEncoder(w).Encode(movimientos)
  if err != nil {
    http.Error(w, "error al enviar los datos del getEgresos", http.StatusInternalServerError)
  }
}

//putEgreso agrega un moviviento en la tabla de tipo egreso, se resive con un Json.
func postEgreso(w http.ResponseWriter, r *http.Request) {
  //Creo la variable para almacenar los datos que envia el cliente
  var m Movimiento
  //Decodifico el dato de un json a la variable creada al mismo tiempo que evaluo el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    http.Error(w, "Error al leer el json.", http.StatusBadRequest)
    return
  }
  //validamos que si engresara los campos obligatorios
  err = comprobarMovimiento(m)
  if err != nil {
    http.Error(w, "Error, datos omitidos en el egreso", http.StatusBadRequest)
    return
  }
  
  //Establesco las variables que se usaran para la manejar los movimientos.
  m.Tipo = "egreso"
  m.Creado = time.Now()
  
  //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO movimientos ( tipo, monto, descripcion, grupo, fecha, creado ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, m.Creado)
  //Valido el error al insertar los datos
  if err != nil {
    http.Error(w, "Error al insertar egreso en la tabla.", http.StatusInternalServerError)
    return
  }
  
  //Establesco la cabecera para responder
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
  //Convertimos la estructura a json y enviamos los datos
  //comprobamos el error al pasarlos
  err = json.NewEncoder(w).Encode(m)
  if err != nil {
    http.Error(w, "Error al escribir el json con los datos que se ingresaron.", http.StatusInternalServerError)
  }
}

//postIngreso agrega a la base de datos un movimiento con el tipo ingreso
func postIngreso(w http.ResponseWriter, r *http.Request) {
  var m Movimiento
  //leemos los datos json y los pasamos a las estructura
  //comprobamos el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    http.Error(w, "Error al leer los datos del json.", http.StatusBadRequest)
    return
  }
  
  //validamos que si ingresara los campos obligatorios
  err = comprobarMovimiento(m)
  if err != nil {
    http.Error(w, "Error, datos de movimiento omitidos.", http.StatusBadRequest)
    return
  }
  
  m.Tipo = "ingreso"
  m.Creado = time.Now()
  
    //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO movimientos ( tipo, monto, descripcion, grupo, fecha, creado ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, m.Creado)
  //Valido el error al insertar los datos
  if err != nil {
    http.Error(w, "Error al insertar ingreso en la tabla.", http.StatusInternalServerError)
    return
  }
  
  //Establesco la cabecera para responder
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
  //Convertimos la estructura a json y enviamos los datos
  //comprobamos el error al pasarlos
  err = json.NewEncoder(w).Encode(m)
  if err != nil {
    http.Error(w, "Error al escribir el json con los datos que se ingresaron.", http.StatusInternalServerError)
  }
}

func main() {
  initDB()
  defer db.Close()
  r := mux.NewRouter()
  
  r.HandleFunc("/egreso", getEgresos).Methods("GET")
  r.HandleFunc("/ingreso", getIngresos).Methods("GET")
  r.HandleFunc("/ingreso", postIngreso).Methods("POST")
  r.HandleFunc("/egreso", postEgreso).Methods("POST")
  r.HandleFunc("/egreso/total", holaMundo).Methods("GET")
  r.HandleFunc("/ingreso/total", holaMundo).Methods("GET")
  r.HandleFunc("/ingreso/{id}", holaMundo).Methods("GET")
  r.HandleFunc("/egreso/{id}", holaMundo).Methods("GET")
  r.HandleFunc("/ingreso/{id}", holaMundo).Methods("PUT")
  r.HandleFunc("/egreso/{id}", holaMundo).Methods("PUT")
  r.HandleFunc("/egreso/{id}", holaMundo).Methods("DELETE")
  r.HandleFunc("/ingreso/{id}", holaMundo).Methods("DELETE")
  r.HandleFunc("/export/{type}", holaMundo).Methods("GET")
  
  
  
  server := http.Server{
    Addr: "100.69.187.16:8080",
    Handler: r,
    WriteTimeout: 10 * time.Second,
    ReadTimeout: 10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  
  log.Println("Listening in http://100.69.187.16:8080...")
  log.Fatal(server.ListenAndServe())
}