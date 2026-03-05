require('dotenv').config();
const { McpServer } = require("@modelcontextprotocol/sdk/server/mcp.js");
const { StdioServerTransport } = require("@modelcontextprotocol/sdk/server/stdio.js");
const { z } = require("zod");
const axios = require("axios");

// Go API backend URL
const API_BASE_URL = process.env.API_BASE_URL || "http://localhost:8080/api";

const server = new McpServer({
    name: "telegram-quiz-bot-mcp",
    version: "1.0.0"
});

// Helper for making API requests
async function makeApiRequest(method, endpoint, data = null) {
    try {
        const url = `${API_BASE_URL}${endpoint}`;
        const config = { method, url, data };
        const response = await axios(config);
        return {
            content: [{ type: "text", text: JSON.stringify(response.data, null, 2) }]
        };
    } catch (error) {
        let errorMessage = error.message;
        if (error.response) {
            errorMessage = `API Error (${error.response.status}): ${JSON.stringify(error.response.data)}`;
        }
        return {
            isError: true,
            content: [{ type: "text", text: errorMessage }]
        };
    }
}

// ==========================================
// Tools Registration
// ==========================================

// get_plan
server.tool(
    "get_plan",
    "Returns the full hierarchy of segments and quizzes from the learning curriculum.",
    {},
    async () => {
        return await makeApiRequest("GET", "/plan");
    }
);

// add_segment
server.tool(
    "add_segment",
    "Adds a new learning segment to the curriculum.",
    {
        title: z.string().describe("The title of the segment"),
        description: z.string().describe("A description of what will be learned"),
        order_index: z.number().describe("The order index for this segment")
    },
    async (params) => {
        return await makeApiRequest("POST", "/segments", params);
    }
);

// update_segment
server.tool(
    "update_segment",
    "Updates an existing learning segment.",
    {
        id: z.number().describe("The ID of the segment to update"),
        title: z.string().describe("The title of the segment"),
        description: z.string().describe("A description of what will be learned"),
        order_index: z.number().describe("The order index for this segment")
    },
    async ({ id, ...data }) => {
        return await makeApiRequest("PUT", `/segments/${id}`, data);
    }
);

// add_quiz
server.tool(
    "add_quiz",
    "Adds a new quiz topic to a learning segment.",
    {
        segment_id: z.number().describe("The ID of the segment to add this quiz to"),
        title: z.string().describe("The title of the quiz topic"),
        description: z.string().describe("A description of the quiz content"),
        order_index: z.number().describe("The order index for this quiz within the segment")
    },
    async ({ segment_id, ...data }) => {
        return await makeApiRequest("POST", `/segments/${segment_id}/quizzes`, data);
    }
);

// update_quiz
server.tool(
    "update_quiz",
    "Updates an existing quiz topic.",
    {
        id: z.number().describe("The ID of the quiz to update"),
        segment_id: z.number().optional().describe("The ID of the segment this quiz belongs to"),
        title: z.string().describe("The title of the quiz topic"),
        description: z.string().describe("A description of the quiz content"),
        order_index: z.number().describe("The order index for this quiz within the segment")
    },
    async ({ id, ...data }) => {
        return await makeApiRequest("PUT", `/quizzes/${id}`, data);
    }
);

// add_question
server.tool(
    "add_question",
    "Adds a new multiple-choice question to a quiz.",
    {
        quiz_id: z.number().describe("The ID of the quiz to add this question to"),
        text: z.string().describe("The question text"),
        options: z.array(z.string()).describe("A list of options for the multiple-choice question"),
        correct_index: z.number().describe("The index (0-based) of the correct option"),
        explanation: z.string().describe("An explanation for the correct answer")
    },
    async ({ quiz_id, ...data }) => {
        return await makeApiRequest("POST", `/quizzes/${quiz_id}/questions`, data);
    }
);

// update_question
server.tool(
    "update_question",
    "Updates an existing multiple-choice question.",
    {
        id: z.number().describe("The ID of the question to update"),
        quiz_id: z.number().optional().describe("The ID of the quiz this question belongs to"),
        text: z.string().describe("The question text"),
        options: z.array(z.string()).describe("A list of options for the multiple-choice question"),
        correct_index: z.number().describe("The index (0-based) of the correct option"),
        explanation: z.string().describe("An explanation for the correct answer")
    },
    async ({ id, ...data }) => {
        return await makeApiRequest("PUT", `/questions/${id}`, data);
    }
);


// ==========================================
// Server Startup
// ==========================================

async function startServer() {
    const transport = new StdioServerTransport();
    await server.connect(transport);
    console.error("MCP Server running on stdio");
}

startServer().catch(console.error);

module.exports = { server, makeApiRequest };