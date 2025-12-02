import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

void main() {
  runApp(const DentalApp());
}

class DentalApp extends StatelessWidget {
  const DentalApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Dental App',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blueAccent),
        useMaterial3: true,
      ),
      home: const PatientListScreen(),
    );
  }
}

class PatientListScreen extends StatefulWidget {
  const PatientListScreen({super.key});

  @override
  State<PatientListScreen> createState() => _PatientListScreenState();
}

class _PatientListScreenState extends State<PatientListScreen> {
  // Aquí guardaremos la lista de pacientes que venga del Backend
  List<dynamic> patients = [];
  bool isLoading = true;
  String? errorMessage;

  @override
  void initState() {
    super.initState();
    fetchPatients();
  }

  // Función mágica que habla con Go
  Future<void> fetchPatients() async {
    try {
      // 1. Hacemos la petición al Backend (Puerto 8080)
      // Nota: En IDX/Web usamos localhost. En emulador Android sería 10.0.2.2
      final url = Uri.parse('http://localhost:8080/api/v1/patients');
      final response = await http.get(url);

      if (response.statusCode == 200) {
        // 2. Si responde OK, decodificamos el JSON
        setState(() {
          patients = json.decode(response.body);
          isLoading = false;
        });
      } else {
        throw Exception('Error del servidor: ${response.statusCode}');
      }
    } catch (e) {
      // 3. Si algo falla (ej: servidor apagado), mostramos error
      setState(() {
        isLoading = false;
        errorMessage = "No se pudo conectar: $e";
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Pacientes Dr. House 🦷'),
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              setState(() {
                isLoading = true;
                errorMessage = null;
              });
              fetchPatients();
            },
          )
        ],
      ),
      body: isLoading
          ? const Center(child: CircularProgressIndicator())
          : errorMessage != null
              ? Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      const Icon(Icons.error_outline, size: 48, color: Colors.red),
                      const SizedBox(height: 10),
                      Text(errorMessage!, textAlign: TextAlign.center),
                      const SizedBox(height: 20),
                      ElevatedButton(
                        onPressed: fetchPatients,
                        child: const Text("Reintentar"),
                      )
                    ],
                  ),
                )
              : patients.isEmpty
                  ? const Center(child: Text("No hay pacientes registrados aún."))
                  : ListView.builder(
                      itemCount: patients.length,
                      itemBuilder: (context, index) {
                        final p = patients[index];
                        return Card(
                          margin: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                          elevation: 2,
                          child: ListTile(
                            leading: CircleAvatar(
                              backgroundColor: Colors.blueAccent,
                              foregroundColor: Colors.white,
                              child: Text(p['first_name'] != null ? p['first_name'][0] : "?"),
                            ),
                            title: Text("${p['first_name']} ${p['last_name']}"),
                            subtitle: Text("Doc: ${p['document_number']}"),
                            trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                            onTap: () {
                              // Aquí luego pondremos la pantalla de detalle
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text("Seleccionaste a ${p['first_name']}")),
                              );
                            },
                          ),
                        );
                      },
                    ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          // Aquí luego pondremos el formulario de crear
        },
        child: const Icon(Icons.add),
      ),
    );
  }
}