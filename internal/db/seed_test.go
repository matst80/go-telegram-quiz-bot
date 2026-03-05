package db

import (
	"encoding/json"
	"testing"
)

func TestSeedDatabase(t *testing.T) {
	// Use the actual database file for seeding
	database, err := New("../../quizbot.db")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	seedData := `[                                                                                                                                                  
      {                                                                                                                                                
        "topic": "Basic Greetings (Hola, Adiós, Buenos días)",                                                                                         
        "question": "Which Spanish phrase means 'Good morning'?",                                                                                      
        "options": ["Buenas noches", "Buenos días", "Buenas tardes", "Adiós"],                                                                         
        "correct_answer": "Buenos días"                                                                                                                
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Basic Greetings (Hola, Adiós, Buenos días)",                                                                                         
        "question": "Which word is used to say goodbye in Spanish?",                                                                                   
        "options": ["Hola", "Por favor", "Adiós", "Gracias"],                                                                                          
        "correct_answer": "Adiós"                                                                                                                      
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Numbers 1 to 10",                                                                                                                            
        "question": "How do you say 'three' in Spanish?",                                                                                              
        "options": ["Uno", "Dos", "Tres", "Cuatro"],                                                                                                   
        "correct_answer": "Tres"                                                                                                                       
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Numbers 1 to 10",                                                                                                                            
        "question": "Which number is 'siete'?",                                                                                                        
        "options": ["6", "7", "8", "9"],                                                                                                               
        "correct_answer": "7"                                                                                                                          
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Colors (Red, Blue, Green, Yellow)",                                                                                                   
        "question": "What is the Spanish word for 'Red'?",                                                                                             
        "options": ["Azul", "Rojo", "Verde", "Amarillo"],                                                                                              
        "correct_answer": "Rojo"                                                                                                                       
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Colors (Red, Blue, Green, Yellow)",                                                                                                   
        "question": "How do you say 'Green' in Spanish?",                                                                                              
        "options": ["Amarillo", "Verde", "Azul", "Rojo"],                                                                                              
        "correct_answer": "Verde"                                                                                                                      
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Family Members (Padre, Madre, Hermano)",                                                                                             
        "question": "What does 'Madre' mean?",                                                                                                         
        "options": ["Father", "Mother", "Brother", "Sister"],                                                                                          
        "correct_answer": "Mother"                                                                                                                     
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Family Members (Padre, Madre, Hermano)",                                                                                             
        "question": "How do you say 'Brother' in Spanish?",                                                                                            
        "options": ["Padre", "Madre", "Hermano", "Hijo"],                                                                                              
        "correct_answer": "Hermano"                                                                                                                    
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Days of the Week",                                                                                                                   
        "question": "Which day is 'Lunes'?",                                                                                                           
        "options": ["Sunday", "Monday", "Tuesday", "Wednesday"],                                                                                       
        "correct_answer": "Monday"                                                                                                                     
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Days of the Week",                                                                                                                   
        "question": "How do you say 'Saturday' in Spanish?",                                                                                           
        "options": ["Viernes", "Sábado", "Domingo", "Jueves"],                                                                                         
        "correct_answer": "Sábado"                                                                                                                     
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Months of the Year",                                                                                                                 
        "question": "Which month is 'Enero'?",                                                                                                         
        "options": ["January", "February", "March", "April"],                                                                                          
        "correct_answer": "January"                                                                                                                    
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Months of the Year",                                                                                                                 
        "question": "How do you say 'May' in Spanish?",                                                                                                
        "options": ["Marzo", "Abril", "Mayo", "Junio"],                                                                                                
        "correct_answer": "Mayo"                                                                                                                       
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Common Animals (Dog, Cat, Bird)",                                                                                                    
        "question": "What is 'Perro' in English?",                                                                                                     
        "options": ["Cat", "Bird", "Dog", "Horse"],                                                                                                    
        "correct_answer": "Dog"                                                                                                                        
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Common Animals (Dog, Cat, Bird)",                                                                                                    
        "question": "How do you say 'Cat' in Spanish?",                                                                                                
        "options": ["Gato", "Pájaro", "Conejo", "Pez"],                                                                                                
        "correct_answer": "Gato"                                                                                                                       
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Basic Foods (Bread, Water, Apple)",                                                                                                  
        "question": "What is 'Pan'?",                                                                                                                  
        "options": ["Water", "Apple", "Bread", "Milk"],                                                                                                
        "correct_answer": "Bread"                                                                                                                      
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Basic Foods (Bread, Water, Apple)",                                                                                                  
        "question": "How do you say 'Water' in Spanish?",                                                                                              
        "options": ["Leche", "Agua", "Jugo", "Vino"],                                                                                                  
        "correct_answer": "Agua"                                                                                                                       
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Basic Verbs (To be, To have, To go)",                                                                                                
        "question": "How do you say 'To be' (permanent) in Spanish?",                                                                                  
        "options": ["Estar", "Ser", "Tener", "Ir"],                                                                                                    
        "correct_answer": "Ser"                                                                                                                        
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Basic Verbs (To be, To have, To go)",                                                                                                
        "question": "What does 'Tener' mean?",                                                                                                         
        "options": ["To go", "To have", "To be", "To do"],                                                                                             
        "correct_answer": "To have"                                                                                                                    
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Telling Time in Spanish",                                                                                                            
        "question": "How do you say 'It is one o'clock'?",                                                                                             
        "options": ["Son las uno", "Es la una", "Es el uno", "Son las una"],                                                                           
        "correct_answer": "Es la una"                                                                                                                  
      },                                                                                                                                               
      {                                                                                                                                                
        "topic": "Telling Time in Spanish",                                                                                                            
        "question": "How do you say 'What time is it?'",                                                                                               
        "options": ["¿Qué hora es?", "¿Cuándo es?", "¿Dónde es?", "¿Quién es?"],                                                                       
        "correct_answer": "¿Qué hora es?"                                                                                                              
      }                                                                                                                                                
    ]`

	var quizzes []Quiz
	if err := json.Unmarshal([]byte(seedData), &quizzes); err != nil {
		t.Fatalf("Failed to unmarshal seed data: %v", err)
	}

	lessons := map[string]string{
		"Basic Greetings (Hola, Adiós, Buenos días)": "Welcome to your first Spanish lesson! 👋\n\n**Key Phrases:**\n- **Hola**: Hello\n- **Adiós**: Goodbye\n- **Buenos días**: Good morning\n- **Buenas tardes**: Good afternoon\n- **Buenas noches**: Good night / Good evening",
		"Numbers 1 to 10":                        "Let's learn to count in Spanish! 🔢\n\n**Numbers:**\n- **Uno**: 1\n- **Dos**: 2\n- **Tres**: 3\n- **Cuatro**: 4\n- **Cinco**: 5\n- **Seis**: 6\n- **Siete**: 7\n- **Ocho**: 8\n- **Nueve**: 9\n- **Diez**: 10",
		"Colors (Red, Blue, Green, Yellow)":      "Spice up your Spanish with colors! 🎨\n\n**Colors:**\n- **Rojo**: Red\n- **Azul**: Blue\n- **Verde**: Green\n- **Amarillo**: Yellow",
		"Family Members (Padre, Madre, Hermano)": "Learn how to talk about your family! 👨‍👩‍👧‍👦\n\n**Common Family Words:**\n- **Padre**: Father\n- **Madre**: Mother\n- **Hermano**: Brother\n- **Hermana**: Sister\n- **Hijo**: Son / **Hija**: Daughter",
		"Days of the Week":                       "Planning your week in Spanish? 📅\n\n**Days:**\n- **Lunes**: Monday\n- **Martes**: Tuesday\n- **Miércoles**: Wednesday\n- **Jueves**: Thursday\n- **Viernes**: Friday\n- **Sábado**: Saturday\n- **Domingo**: Sunday",
		"Months of the Year":                     "Learn the months in Spanish! 🗓️\n\n**Months:**\n- **Enero**: January\n- **Mayo**: May\n- **Septiembre**: September\n- **Diciembre**: December",
		"Common Animals (Dog, Cat, Bird)":        "Talk about your pets! 🐕🐈\n\n**Animals:**\n- **Perro**: Dog\n- **Gato**: Cat\n- **Pájaro**: Bird",
		"Basic Foods (Bread, Water, Apple)":      "Ordering food is essential! 🍎🍞\n\n**Foods:**\n- **Pan**: Bread\n- **Agua**: Water\n- **Manzana**: Apple",
		"Basic Verbs (To be, To have, To go)":    "Verbs are the engine of sentences! ⚙️\n\n**Verbs:**\n- **Ser/Estar**: To be\n- **Tener**: To have\n- **Ir**: To go",
		"Telling Time in Spanish":                "¿Qué hora es? (What time is it?) ⌚\n\n**Time Phrases:**\n- **Es la una**: It is 1:00\n- **Son las dos**: It is 2:00",
	}

	for topic, content := range lessons {
		if err := database.SaveLesson(topic, content); err != nil {
			t.Errorf("Failed to save lesson for topic '%s': %v", topic, err)
		}
	}

	for _, q := range quizzes {
		id, err := database.SaveQuiz(q)
		if err != nil {
			t.Errorf("Failed to save quiz '%s': %v", q.Question, err)
			continue
		}
		t.Logf("Saved quiz with ID: %d", id)
	}
}
