package main

import (
  "net/http"
  "time"
  "fmt"
  "database/sql"
  "encoding/json"
  "encoding/csv"
  "log"
  "strconv"
  
  _ "modernc.org/sqlite"
  "github.com/gorilla/mux"
  "github.com/joho/godotenv"
)

var db *sql.DB

//GETS

//getEgresos consulta los egresos en la tabla, que sean egresos y luego
//los envia en formato json al navegador.
func getEgresos(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //consultamos en la tabla los egresos
  registros, err := getRegistros("egreso", nombreUsuario)
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
    return
  }
  
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slice a json y los enviamos
  //al usuario, valodamos el error
  json.NewEncoder(w).Encode(registros)
}

//getIngresos consulta en la base de datos y retorna los ingresos ssegun el usuario.
func getIngresos(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //consultamos los movimientos tipo ingreso, validamos el error.
  registros, err := getRegistros("ingreso", nombreUsuario)
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
    return
  }
  
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
 
  //Pasamos todos los datos del slice a json y los enviamos
  //al usuario, valodamos el error
  err = json.NewEncoder(w).Encode(registros)
  if err != nil {
    http.Error(w, "error al enviar los datos del getIngresos", http.StatusInternalServerError)
  }
}

//getTotalEgresos devuelve el total de egresos dependiendo de las fechas
//que se le pasen, sumara todo entre ellas y devolvera solo la suma
//Ejm consulta http://100.69.187.16:8080/totalEgresos?desde=2024-12-20T00:00:00Z&hasta=2024-12-31T00:00:00Z
func getTotalEgresos(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Recibe las fechas y en el formato para time.Time y validamos el error.
  desde, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("desde"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'desde', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  hasta, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("hasta"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'hasta', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //consultamos en la tabla egresos con fecha de inicio y fin.
  total, err := getTotal("egreso", desde, hasta, nombreUsuario)
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
    return
  }
  
  // Devolvemos el total en JSON, lo pasamos a map ya que la funcion Encode
  //necesita un tipo de dato que sea compatiple para codificar.
  jsonTotalEgresos := map[string]int{"total": total}
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(jsonTotalEgresos)
}

//getTotalIngresos devuelve el total de ingresos dependiendo de las fechas
//que se le pasen, sumara todo entre ellas y devolvera solo la suma
//consulta http://100.69.187.16:8080/totalIngresos?desde=2024-12-04T00:00:00Z&hasta=2024-12-20T00:00:00Z
func getTotalIngresos(w http.ResponseWriter, r *http.Request) {
  
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Convertimos los strings a formato fecha y validamos los errores
  desde, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("desde"))
  if err != nil {
    errorStr := fmt.Sprintf("Error al ingresae la fecha, %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  hasta, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("hasta"))
  if err != nil {
    errorStr := fmt.Sprintf("Error al ingresae la fecha, %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //consultamos en la tabla solos los ingresos entre las fechas.
  total, err := getTotal("ingreso", desde, hasta, nombreUsuario)
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
    return
  }
  
  // Devolvemos el total en JSON, lo pasamos a map ya que la funcion Encode
  //necesita un tipo de dato que sea compatiple para codificar.
  jsonTotalIngresos := map[string]int{"total": total}
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(jsonTotalIngresos)
}

//getById retorna un moviviento dependiendo del id y usuario que se pasa como
//variqble en la URL. ejm: http://100.69.187.16:8080/movimiento/10
func getById(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Sacamos la variable.
  //validamos que sea de tipo int
  id, err := strconv.Atoi(mux.Vars(r)["id"])
  if err != nil {
    http.Error(w, "Error en id, se esperaba un numero de tipo int.", http.StatusBadRequest)
    return
  }
  
  //consultamos la tabla con id y usuario.
  m, err := getRegistroById(id, nombreUsuario)
  if err != nil {
    writeError(w, "Error en al consultar el registro", err, http.StatusInternalServerError)
    return
  }
  
  //establecemos cabeceras y respondemos con un json.
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(m)
}

//exportFechas exporta a un archivo .json o .csv despendiendo del tipo 
//dado solo los regustris que esten dentro del rango de fechas que se le pase.
//ejm http://10.151.44.98:8080/exportRango?desde=2024-12-04T00:00:00Z&hasta=2024-12-20T00:00:00Z&tipo=json
func exportFechas(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Recibe las fechas y en el formato para time.Time y validamos el error.
  desde, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("desde"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'desde', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  hasta, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("hasta"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'hasta', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  //obtenemos el tipo de archivo y validamos que solo sean los que maneja
  tipo := string(r.URL.Query().Get("tipo"))
  if tipo != "json" && tipo != "csv" {
    http.Error(w, "El tipo solo puede ser json o csv.", http.StatusBadRequest)
    return
  }
  
  //Consultamos la base de datis, validamos el error.
  registros, err := getRegistrosFechas(desde, hasta, nombreUsuario)
  if err != nil {
    writeError(w, "Error al consultar los regustris en la base de datos.", err, http.StatusInternalServerError)
    return
  }
  
	if tipo == "csv" {
	  
    //establecemos las cabeceras para indicar que es una descarga de archivo CSV
	  w.Header().Set("Content-Disposition", "attachment; filename=registros.csv")
	  //le informamos el tipo de arcrivo que sera.
	  w.Header().Set("Content-Type", "text/csv")
    
    //creamos un writer del resposeWriter para poder escribirle
    writer := csv.NewWriter(w)
    defer writer.Flush() //Lo liberamos
    
    // Encabezado para el archivo
	  writer.Write([]string{"Tipo", "Monto", "Descripcion", "Grupo", "Fecha"})
    
    //Recirremos el slite de movimientos para imprimirlos en cada fila del csv  
    for _, fila := range registros {
      //La funcion movimientoASlice pasa cada estructura tupo Registro a
      //un slite de string con solo los campos de tipo, monto, Descripcion, grupo y fecha
      err = writer.Write(movimientoASlice(fila))
      if err != nil {
        errorStr := fmt.Sprintf("Error al escribir el el archivo. %v", err)
        http.Error(w, errorStr, http.StatusInternalServerError)
        return
      }
    }
	} else {
    //establecemos las cabeceras para indicar que es una descarga de archivo CSV
	  w.Header().Set("Content-Disposition", "attachment; filename=registros.json")
	  //le informamos el tipo de arcrivo que sera.
	  w.Header().Set("Content-Type", "application/json")
  	
  	//Creamos un Encoder del archivo para escribir formato json en el.
  	encoder := json.NewEncoder(w)
  	encoder.SetIndent("", "  ")
  	//escribimos todo el slite de movimientos
  	err = encoder.Encode(registrosASimples(registros))
  	if err != nil {
  	  http.Error(w, "Error al escribir en el archivo", http.StatusInternalServerError)
  	  return
  	}
	}
  
}

//POSTS

//postEgreso agrega un moviviento en la tabla de tipo egreso, se resive con un Json.
//Json ejemplo{"monto": 22,"fecha": "2024-12-05T00:00:00Z"}
func postEgreso(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Creo la variable para almacenar los datos que envia el cliente
  var m Registro
  //Decodifico el dato de un json a la variable creada al mismo tiempo que evaluo el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    http.Error(w, "Error al leer el json.", http.StatusBadRequest)
    return
  }
  //validamos que si engresara los campos obligatorios
  err = comprobarInfoRequest(m)
  if err != nil {
    http.Error(w, "Error, datos omitidos en el egreso", http.StatusBadRequest)
    return
  }
  
  //Establesco las variables que se usaran para la manejar los movimientos.
  m.Tipo = "egreso"
  
  //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO registros ( tipo, monto, descripcion, grupo, fecha, usuario ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, nombreUsuario)
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
    errorStr := fmt.Sprintf("Error al escribir el json con los datos que se ingresaron. %v", err)
    http.Error(w, errorStr, http.StatusInternalServerError)
  }
}

//postIngreso agrega a la base de datos un movimiento con el tipo ingreso
func postIngreso(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  var m Registro
  //leemos los datos json y los pasamos a las estructura
  //comprobamos el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al leer los datos del json. %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //validamos que si ingresara los campos obligatorios
  err = comprobarInfoRequest(m)
  if err != nil {
    http.Error(w, "Error, datos de movimiento omitidos.", http.StatusBadRequest)
    return
  }
  
  m.Tipo = "ingreso"
  
  //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO registros ( tipo, monto, descripcion, grupo, fecha, usuario ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, nombreUsuario)
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

func registrar(w http.ResponseWriter, r *http.Request) {
  var u Usuario
  err := json.NewDecoder(r.Body).Decode(&u)
  if err != nil {
    writeError(w, "Error al obtener los datos del body", err, http.StatusBadRequest)
  }
  
  err = guardarUsuario(u)
  if err != nil {
    writeError(w, "Error al guardar el usuario y clave.", err, http.StatusInternalServerError)
  }
  
  w.WriteHeader(http.StatusCreated)
  
}

func login(w http.ResponseWriter, r *http.Request) {
  var u Usuario
  err := json.NewDecoder(r.Body).Decode(&u)
  if err != nil {
    writeError(w, "Error al obtener los datos del body", err, http.StatusBadRequest)
  }
  
  err = comprobarUsuario(u)
  if err != nil {
    writeError(w, "Error el usuario o contraseña incorecto.", err, http.StatusInternalServerError)
    return
  }
  
  tokenString, err := crearJWT(u.Nombre)
  if err != nil {
    writeError(w, "Error al crear el jwt.", err, http.StatusInternalServerError)
    return
  }
  
  //Enviamos el token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

//PUTS

//putById actualiza un registro en la tabla segun el id que se pase como
//variable por URL con los datos tipo json a travez del body. De momento se
//asume que el cliente envia los datos completos.
//ejm http://100.69.187.16:8080/movimiento/9
// {"monto": 333, "grupo": "nuevo", "usuario": "carlos"}
//ToDo: LOS DATOS OMITIDOS DEJARLOS CON EL MISMO VALOR.
func putById(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Extraemos la el id de la URL y aseguramos que sea un int.
  id, err := strconv.Atoi(mux.Vars(r)["id"])
  if err != nil {
    http.Error(w, "Error en id, se esperaba un numero de tipo int.", http.StatusBadRequest)
    return
  }
  
  //Variable para extraer los datos a actualizar.
  //De momenro los actualiza asumiendo que pasa todos los datos.
  var m Registro
  //Pasamos los datos a la variable y comprobamos el error.
  err = json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al leer los datos del json. %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //Actualizamos los datos en la tabla por id y validamos el error.
  _, err = db.Exec("UPDATE registros SET monto = ?, descripcion = ?, grupo = ?, fecha = ? WHERE id = ? AND usuario = ?", m.Monto, m.Descripcion, m.Grupo, m.Fecha, id, nombreUsuario)
  if err != nil {
    errorStr := fmt.Sprintf("Error al actualizar el registro en la base de datos con el id ingresado. %v", err)
    http.Error(w, errorStr, http.StatusInternalServerError)
    return
  }
  
  //establecemos cabeceras y respondemos con un json.
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(m)
}

//DELETES

//deleteById elimina un registro segun el id ingresado.
func deleteById(w http.ResponseWriter, r *http.Request) {
  nombreUsuario := r.Context().Value("usuario").(string)
  
  //Extraemos la el id de la URL y aseguramos que sea un int.
  id, err := strconv.Atoi(mux.Vars(r)["id"])
  if err != nil {
    http.Error(w, "Error en id, se esperaba un numero de tipo int.", http.StatusBadRequest)
    return
  }
  
  //preparamos la instruccion para sqlite.
  stmt, err := db.Prepare("DELETE FROM registros WHERE id = ? AND usuario = ?")
  if err != nil {
    http.Error(w, "Error preparando SQL", http.StatusInternalServerError)
    return
  }
  //cerramos la base de datos.
  defer stmt.Close()
  
  //Ejecutamos la instruccion para eliminar el regustri.
  res, err := stmt.Exec(id, nombreUsuario)
  if err != nil {
    http.Error(w, "Error ejecutando DELETE", http.StatusInternalServerError)
    return
  }
  
  //Validamos que si elimine el regustro
  filas, err := res.RowsAffected()
  if err != nil || filas == 0 {
    http.Error(w, "No se eliminó ningún registro", http.StatusNotFound)
    return
  }
  
  //respondemos al usuario.
  w.WriteHeader(http.StatusOK)
  fmt.Fprintf(w, "Movimiento con ID %s eliminado correctamente", id)
}

func main() {
  //Cargamos el godotenv para poder ver la ckave secreta que gebera el jwt
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error cargando .env")
  }
  
  initDB()
  defer db.Close()
  r := mux.NewRouter()
  
  r.Handle("/egreso", authMiddleware(http.HandlerFunc(getEgresos))).Methods("GET")
  r.Handle("/ingreso", authMiddleware(http.HandlerFunc(getIngresos))).Methods("GET")
  r.Handle("/totalEgresos", authMiddleware(http.HandlerFunc(getTotalEgresos))).Methods("GET")
  r.Handle("/totalIngresos", authMiddleware(http.HandlerFunc(getTotalIngresos))).Methods("GET")
  r.Handle("/movimiento/{id}", authMiddleware(http.HandlerFunc(getById))).Methods("GET")
  r.Handle("/exportRango", authMiddleware(http.HandlerFunc(exportFechas))).Methods("GET")
  r.Handle("/ingreso", authMiddleware(http.HandlerFunc(postIngreso))).Methods("POST")
  r.Handle("/egreso", authMiddleware(http.HandlerFunc(postEgreso))).Methods("POST")
  r.HandleFunc("/registrar", registrar).Methods("POST")
  r.HandleFunc("/login", login).Methods("POST")
  r.Handle("/movimiento/{id}", authMiddleware(http.HandlerFunc(putById))).Methods("PUT")
  r.Handle("/movimiento/{id}", authMiddleware(http.HandlerFunc(deleteById))).Methods("DELETE")
  
  
  
  server := http.Server{
    Addr: "10.254.97.246:8080",
    Handler: r,
    WriteTimeout: 10 * time.Second,
    ReadTimeout: 10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  
  log.Println("Listening in http://10.254.97.246:8080...")
  log.Fatal(server.ListenAndServe())
}