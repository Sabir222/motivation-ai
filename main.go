package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type GeminiRequest struct {
	Contents []Content `json:"contents"`
}
type Content struct {
	Parts []Part `json:"parts"`
}
type Part struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}
type Candidate struct {
	Content       Content        `json:"content"`
	FinishReason  string         `json:"finishReason"`
	Index         int            `json:"index"`
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

const geminiApiUrlBase = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent"

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not load .env file:", err)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		// Make the error message clearer
		log.Fatal("FATAL: GEMINI_API_KEY environment variable not set (checked .env file and environment).")
	}

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/generate", generateHandler)

	// Start server
	port := os.Getenv("PORT") // Also reads PORT from .env if set
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

// generateHandler handles the form submission and calls Gemini API
func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Error processing request", http.StatusBadRequest)
		return
	}

	yourName := r.FormValue("your_name")
	position := r.FormValue("position")
	companyName := r.FormValue("company_name")
	keySkills := r.FormValue("key_skills")
	whyInterested := r.FormValue("why_interested")
	relevantExperience := r.FormValue("relevant_experience")

	if yourName == "" || position == "" || companyName == "" || whyInterested == "" {
		log.Println("Validation Error: Missing required fields")
		http.Error(w, "<p class='text-red-600'>Please fill in all required fields (Name, Position, Company, Why Interested).</p>", http.StatusBadRequest)
		return
	}

	prompt := buildPrompt(yourName, position, companyName, keySkills, whyInterested, relevantExperience)
	log.Printf("Generated Prompt: %s\n", prompt)

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("Error: GEMINI_API_KEY not configured on server (checked .env and environment).")
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	generatedText, err := callGeminiAPI(apiKey, prompt)
	if err != nil {
		log.Printf("Error calling Gemini API: %v", err)
		http.Error(w, fmt.Sprintf("<p class='text-red-600'>Error generating letter: %v</p>", err), http.StatusInternalServerError)
		return
	}

	htmlResponse := "<h2 class='text-xl font-semibold mb-4'>Generated Motivation Letter:</h2>"
	htmlResponse += "<div class='whitespace-pre-wrap'>"
	htmlResponse += strings.ReplaceAll(generatedText, "\n", "<br />")
	htmlResponse += "</div>"

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, htmlResponse)
	log.Println("Successfully generated and returned letter.")
}

func buildPrompt(name, position, company, skills, interest, experience string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate a professional and enthusiastic motivation letter for %s applying for the position of %s at %s.\n\n", name, position, company))
	sb.WriteString("Key information to include:\n")
	sb.WriteString(fmt.Sprintf("- Applicant's Name: %s\n", name))
	sb.WriteString(fmt.Sprintf("- Position Applying For: %s\n", position))
	sb.WriteString(fmt.Sprintf("- Company Name: %s\n", company))
	if skills != "" {
		sb.WriteString(fmt.Sprintf("- Highlight these key skills: %s\n", skills))
	}
	sb.WriteString(fmt.Sprintf("- Reason for interest in the position/company: %s\n", interest))
	if experience != "" {
		sb.WriteString(fmt.Sprintf("- Briefly mention this relevant experience: %s\n", experience))
	}
	sb.WriteString("\nInstructions:\n")
	sb.WriteString("- The tone should be professional, confident, and genuinely enthusiastic.")
	sb.WriteString("- Structure the letter logically (introduction, body paragraphs highlighting skills/interest/experience, conclusion).")
	sb.WriteString("- Ensure the letter flows well and is grammatically correct.")
	sb.WriteString("- Do not include placeholders like '[Your Name]' or '[Company Name]' in the final letter; use the provided information directly.")
	sb.WriteString("Take inspiration from the following example, adapting the tone, structure, and professionalism, but generate original content tailored to the new applicant:\n\n")

	sb.WriteString("----- BEGIN INSPIRATION LETTER -----\n")
	sb.WriteString("Sabir Koutabi\n")
	sb.WriteString("sabirkoutabi@gmail.com | sabirkoutabi.tech | linkedin.com/in/skoutabi | x.com/sabirkoutabi\n")
	sb.WriteString("* today's date here *, 2025\n")
	sb.WriteString("Lena Berger\n")
	sb.WriteString("Senior Technical Recruiter\n")
	sb.WriteString("Netways GmbH\n")

	sb.WriteString("Subject: Motivation Letter - Application for Front-End Web Developer\n\n")

	sb.WriteString("Dear Ms. Berger,\n\n")

	sb.WriteString("I am writing to express my strong interest in the Front-End Web Developer position at Netways GmbH, as advertised on your company’s careers page. ")
	sb.WriteString("My journey from a detail-oriented background in finance at CIH Bank to a dedicated software engineer, trained through the rigorous ALX Africa/Holberton School program, ")
	sb.WriteString("provides me with a unique blend of analytical thinking and technical proficiency. ")
	sb.WriteString("My specialization in front-end development, complemented by full-stack capabilities (including technologies like TypeScript, GO, Node.js, SQL/PostgreSQL, and API design), ")
	sb.WriteString("has been honed through practical application in my recent internship at Yutapp and ongoing freelance projects, where I build robust web solutions for clients and personal development.\n\n")

	sb.WriteString("During my time at ALX/Holberton, I was immersed in real-world team projects, Agile methodologies, and problem-solving under pressure. ")
	sb.WriteString("One of my proudest achievements was leading the front-end architecture of a real estate web app using Next.js, where I implemented OAuth authentication, ")
	sb.WriteString("integrated MongoDB with Prisma, and designed a responsive, user-first interface using TailwindCSS. ")
	sb.WriteString("These projects strengthened my ability to collaborate, communicate technical ideas, and deliver production-ready code efficiently.\n\n")

	sb.WriteString("My experience as a freelance Web Developer has cultivated strong self-discipline, adaptability, and a results-oriented approach, ")
	sb.WriteString("making me well-suited for contract or freelance engagements where I can quickly integrate and contribute effectively. ")
	sb.WriteString("I am confident in my ability to deliver high-quality work and am open to discussing a trial period to demonstrate my skills and ensure a strong fit with your team's needs and workflow. ")
	sb.WriteString("I am excited about the possibility of bringing my skills and enthusiasm to Netways GmbH and welcome the opportunity to discuss my application further.\n\n")

	sb.WriteString("Thank you for your time and consideration.\n\n")
	sb.WriteString("Sincerely,\n")
	sb.WriteString("Sabir Koutabi\n")
	sb.WriteString("----- END INSPIRATION LETTER -----\n\n")
	sb.WriteString("- Again Do not include placeholders like '[Your Name]' or '[Company Name]' in the final letter; use the provided information directly.")
	return sb.String()
}

func callGeminiAPI(apiKey, prompt string) (string, error) {
	apiURL := fmt.Sprintf("%s?key=%s", geminiApiUrlBase, url.QueryEscape(apiKey))

	reqBody := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshalling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to Gemini API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading Gemini API response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Gemini API Error Response Status: %s", resp.Status)
		log.Printf("Gemini API Error Response Body: %s", string(respBody))
		return "", fmt.Errorf("gemini API request failed with status %s", resp.Status)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		log.Printf("Raw Gemini Response: %s", string(respBody))
		return "", fmt.Errorf("error unmarshalling Gemini API response: %w", err)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	log.Printf("Could not extract text. Raw Gemini Response: %s", string(respBody))
	return "", fmt.Errorf("no content found in Gemini API response")
}
