export type Base = {
    id: string;
    created_at: string;
    updated_at: string;
};
export type Profile = Base & {
 time_zone:string;
 latest_cv_upload_id:string|null;cv_fingerprint:string;
 summary:string;cv_source:string;
    name: string;
    years_experience: number;
    location: string;
    skills: string[];
};
export type Preferences = Base & {
 nearby_locations:string[];hard_location:boolean;require_location_match:boolean;
    roles: string[];
    required_skills: string[];
    seniorities: string[];
    countries: string[];
    remote: string[];
    employment_types: string[];
    excluded_companies: string[];
    excluded_keywords: string[];
    salary_min: number;
    salary_max: number;
    currency: string;
    salary_period: string;
    max_age_days: number;
    hard_remote: boolean;
    weights: Record<string, number>;
    discovery_minutes: number;
    auto_analyse: boolean;
    ai_enabled: boolean;
    ollama_model: string;
    ai_context: number;
    ai_keep_alive: string;
};
export type Analysis = Base & {
    model: string;
    prompt_version: string;
    input_tokens: number;
    output_tokens: number;
    estimated_cost: number | null;
    result: {
        overall_score: number;
        technical_score: number;
        matching_skills: string[];
        missing_skills: string[];
        experience_compatibility: string;
        location_compatibility: string;
        salary_compatibility: string;
        advantages: string[];
        disadvantages: string[];
        concerns: string[];
        unknowns: string[];
        recommendation: string;
        confidence: string;
        explanation: string;
    };
};
export type Job = Base & {
    work_hours?: {target_zone:string;target_label:string;target_location:string;status:'stated'|'example'|'unknown';source:string;note:string;examples:{zone:string;local_hours:string;your_hours:string;difference:string}[]};
 eligible_countries:string[]|null;worldwide:boolean|null;
    deterministic?: Deterministic;
    source_name: string;
    source_url: string;
    application_url: string;
    expires_at: string | null;
    company_id: string;
    company_name: string;
    title: string;
    description: string;
    location: string;
    remote: string;
    salary_min: number | null;
    salary_max: number | null;
    currency: string;
    salary_period: string;
    canonical_url: string;
    first_seen: string;
    last_seen: string;
    posted_at: string | null;
    available: boolean;
    analysis?: Analysis;
    eligibility?: {
        eligible: boolean;
        exclusions: string[];
        preferences: string[];
        unknowns: string[];
    };
    ranking?: {
        score: number | null;
        coverage: number;
        workplace_risk: string;
    };
};
export type JobPage = {
    items: Job[];
    total: number;
    page: number;
    limit: number;
};
export type CompanyFact={value:string;source_url:string;source_name:string;notes:string;recorded_at?:string|null};
export type CompanyProfile=Partial<Record<"summary"|"legal_name"|"origin_country"|"headquarters"|"ceo"|"ceo_nationality",CompanyFact>>;
export type Company = Base & {
 profile:CompanyProfile|null;
    name: string;
    website: string;
    research_urls: string[];
};
export type Source = Base & {
 country:string;
    company_id: string;
    provider: string;
    board: string;
    enabled: boolean;
};
export type Evidence = Base & {
    source_url: string;
    source_name: string;
    published_at: string | null;
    retrieved_at: string;
    summary: string;
    category: string;
    confidence: string;
};
export type Research = Base & {
    status: string;
    summary: string;
    errors: string[];
    expires_at: string;
    interpretation?: {
        summary: string;
        unknowns: string[];
        source_ids: string[];
        label: string;
    };
};
export type Application = Base & {
    job_id: string;
    stage: string;
    notes: string;
};
export type Run = Base & {
    status: string;
    jobs_seen: number;
    errors: string[];
    finished_at: string | null;
};
export type Deterministic = {
    technical_score: number | null;
    matching_skills: string[];
    missing_skills: string[];
    unknowns: string[];
    factors: {
        name: string;
        status: string;
        explanation: string;
        score: number | null;
    }[];
    recommendation: string;
    confidence: string;
};
export type Task = Base & {
    resource_id: string;
    kind: string;
    status: string;
    error: string;
    attempts: number;
};

export type CVAnalysis={name:string;location:string;years_experience:number|null;skills:string[];summary:string;unknowns:string[]};
export type CVUpload=Base&{filename:string;fingerprint:string;model:string;prompt_version:string;ai_context:number;analysis:CVAnalysis|null;applied_at:string|null};
export type CVStatus={upload:CVUpload|null;task:Task|null};
